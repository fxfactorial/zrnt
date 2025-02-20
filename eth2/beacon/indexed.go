package beacon

import (
	"errors"
	"fmt"
	"github.com/protolambda/zrnt/eth2/util/bls"
	"github.com/protolambda/ztyp/codec"
	"github.com/protolambda/ztyp/tree"
	. "github.com/protolambda/ztyp/view"
	"sort"
)

type CommitteeIndices []ValidatorIndex

func (p *CommitteeIndices) Deserialize(spec *Spec, dr *codec.DecodingReader) error {
	return dr.List(func() codec.Deserializable {
		i := len(*p)
		*p = append(*p, ValidatorIndex(0))
		return &((*p)[i])
	}, ValidatorIndexType.TypeByteLength(), spec.MAX_VALIDATORS_PER_COMMITTEE)
}

func (a CommitteeIndices) Serialize(_ *Spec, w *codec.EncodingWriter) error {
	return w.List(func(i uint64) codec.Serializable {
		return a[i]
	}, ValidatorIndexType.TypeByteLength(), uint64(len(a)))
}

func (a CommitteeIndices) ByteLength(*Spec) uint64 {
	return ValidatorIndexType.TypeByteLength() * uint64(len(a))
}

func (*CommitteeIndices) FixedLength(*Spec) uint64 {
	return 0
}

func (p CommitteeIndices) HashTreeRoot(spec *Spec, hFn tree.HashFn) Root {
	return hFn.Uint64ListHTR(func(i uint64) uint64 {
		return uint64(p[i])
	}, uint64(len(p)), spec.MAX_VALIDATORS_PER_COMMITTEE)
}

func (c *Phase0Config) CommitteeIndices() ListTypeDef {
	return ListType(ValidatorIndexType, c.MAX_VALIDATORS_PER_COMMITTEE)
}

type IndexedAttestation struct {
	AttestingIndices CommitteeIndices `json:"attesting_indices" yaml:"attesting_indices"`
	Data             AttestationData  `json:"data" yaml:"data"`
	Signature        BLSSignature     `json:"signature" yaml:"signature"`
}

func (p *IndexedAttestation) Deserialize(spec *Spec, dr *codec.DecodingReader) error {
	return dr.Container(spec.Wrap(&p.AttestingIndices), &p.Data, &p.Signature)
}

func (a *IndexedAttestation) Serialize(spec *Spec, w *codec.EncodingWriter) error {
	return w.Container(spec.Wrap(&a.AttestingIndices), &a.Data, &a.Signature)
}

func (a *IndexedAttestation) ByteLength(spec *Spec) uint64 {
	return codec.ContainerLength(spec.Wrap(&a.AttestingIndices), &a.Data, &a.Signature)
}

func (*IndexedAttestation) FixedLength(*Spec) uint64 {
	return 0
}

func (p *IndexedAttestation) HashTreeRoot(spec *Spec, hFn tree.HashFn) Root {
	return hFn.HashTreeRoot(spec.Wrap(&p.AttestingIndices), &p.Data, p.Signature)
}

func (c *Phase0Config) IndexedAttestation() *ContainerTypeDef {
	return ContainerType("IndexedAttestation", []FieldDef{
		{"attesting_indices", c.CommitteeIndices()},
		{"data", AttestationDataType},
		{"signature", BLSSignatureType},
	})
}

func (spec *Spec) ValidateIndexedAttestationIndicesSet(indexedAttestation *IndexedAttestation) (ValidatorSet, error) {
	// wrap it in validator-sets. Does not sort it, but does make checking if it is a lot easier.
	indices := ValidatorSet(indexedAttestation.AttestingIndices)

	// Verify max number of indices
	if count := uint64(len(indices)); count > spec.MAX_VALIDATORS_PER_COMMITTEE {
		return nil, fmt.Errorf("invalid indices count in indexed attestation: %d", count)
	}

	// empty attestation
	if len(indices) <= 0 {
		return nil, errors.New("in phase 0 no empty attestation signatures are allowed")
	}

	// The indices must be sorted
	if !sort.IsSorted(indices) {
		return nil, errors.New("attestation indices are not sorted")
	}

	// Verify if the indices are unique. Simple O(n) check, since they are already sorted.
	for i := 1; i < len(indices); i++ {
		if indices[i-1] == indices[i] {
			return nil, fmt.Errorf("attestation indices at %d and %d are duplicate, both: %d", i-1, i, indices[i])
		}
	}
	return indices, nil
}

func (spec *Spec) ValidateIndexedAttestationNoSignature(state *BeaconStateView, indexedAttestation *IndexedAttestation) error {
	indices, err := spec.ValidateIndexedAttestationIndicesSet(indexedAttestation)
	if err != nil {
		return err
	}

	// Check the last item of the sorted list to be a valid index,
	// if this one is valid, the others are as well, since they are lower.
	valid, err := state.IsValidIndex(indices[len(indices)-1])
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("attestation indices contain out of range index")
	}
	return nil
}

func (spec *Spec) ValidateIndexedAttestationSignature(dom BLSDomain, pubCache *PubkeyCache, indexedAttestation *IndexedAttestation) error {
	pubkeys := make([]*CachedPubkey, 0, len(indexedAttestation.AttestingIndices))
	for _, i := range indexedAttestation.AttestingIndices {
		pub, ok := pubCache.Pubkey(i)
		if !ok {
			return fmt.Errorf("could not find pubkey for index %d", i)
		}
		pubkeys = append(pubkeys, pub)
	}
	// empty attestation. (Double check, since this function is public, the user might not have validated if it's empty or not)
	if len(pubkeys) <= 0 {
		return errors.New("in phase 0 no empty attestation signatures are allowed")
	}

	if !bls.FastAggregateVerify(pubkeys,
		ComputeSigningRoot(indexedAttestation.Data.HashTreeRoot(tree.GetHashFn()), dom),
		indexedAttestation.Signature,
	) {
		return errors.New("could not verify BLS signature for indexed attestation")
	}
	return nil
}

// Verify validity of slashable_attestation fields.
func (spec *Spec) ValidateIndexedAttestation(epc *EpochsContext, state *BeaconStateView, indexedAttestation *IndexedAttestation) error {
	if err := spec.ValidateIndexedAttestationNoSignature(state, indexedAttestation); err != nil {
		return err
	}
	dom, err := state.GetDomain(spec.DOMAIN_BEACON_ATTESTER, indexedAttestation.Data.Target.Epoch)
	if err != nil {
		return err
	}
	return spec.ValidateIndexedAttestationSignature(dom, epc.PubkeyCache, indexedAttestation)
}
