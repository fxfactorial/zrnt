package epoch_processing

import (
	"context"
	"fmt"
	. "github.com/protolambda/zrnt/eth2/beacon"
	"github.com/protolambda/zrnt/tests/spec/test_util"
	"strings"
	"testing"
)

type RewardsTest struct {
	Spec   *Spec
	Pre    *BeaconStateView
	Input  *RewardsAndPenalties
	Output *RewardsAndPenalties
}

func (c *RewardsTest) ExpectingFailure() bool {
	return false
}

func (c *RewardsTest) Load(t *testing.T, readPart test_util.TestPartReader) {
	c.Spec = readPart.Spec()

	c.Pre = test_util.LoadState(t, "pre", readPart)

	c.Input = &RewardsAndPenalties{}
	sourceDeltas := new(Deltas)
	if test_util.LoadSpecObj(t, "source_deltas", sourceDeltas, readPart) {
		c.Input.Source = sourceDeltas
	} else {
		t.Fatalf("failed to load source_deltas")
	}
	targetDeltas := new(Deltas)
	if test_util.LoadSpecObj(t, "target_deltas", targetDeltas, readPart) {
		c.Input.Target = targetDeltas
	} else {
		t.Fatalf("failed to load target_deltas")
	}
	headDeltas := new(Deltas)
	if test_util.LoadSpecObj(t, "head_deltas", headDeltas, readPart) {
		c.Input.Head = headDeltas
	} else {
		t.Fatalf("failed to load head_deltas")
	}
	inclusionDelayDeltas := new(Deltas)
	if test_util.LoadSpecObj(t, "inclusion_delay_deltas", inclusionDelayDeltas, readPart) {
		c.Input.InclusionDelay = inclusionDelayDeltas
	} else {
		t.Fatalf("failed to load inclusion_delay_deltas")
	}
	inactivityPenaltyDeltas := new(Deltas)
	if test_util.LoadSpecObj(t, "inactivity_penalty_deltas", inactivityPenaltyDeltas, readPart) {
		c.Input.Inactivity = inactivityPenaltyDeltas
	} else {
		t.Fatalf("failed to load inactivity_penalty_deltas")
	}
}

func (c *RewardsTest) Check(t *testing.T) {
	count := uint64(len(c.Input.Source.Rewards))
	diffDeltas := func(name string, computed *Deltas, expected *Deltas) {
		t.Run(name, func(t *testing.T) {
			var failed bool
			var buf strings.Builder
			for i := uint64(0); i < count; i++ {
				if computed.Rewards[i] != expected.Rewards[i] {
					buf.WriteString(fmt.Sprintf("(%s) invalid reward: i: %d, expected: %d, got: %d\n",
						name, i, expected.Rewards[i], computed.Rewards[i]))
					failed = true
				}
				if computed.Penalties[i] != expected.Penalties[i] {
					buf.WriteString(fmt.Sprintf("(%s) invalid penalty: i: %d, expected: %d, got: %d\n",
						name, i, expected.Penalties[i], computed.Penalties[i]))
					failed = true
				}
			}
			if failed {
				t.Error("rewards error:\n" + buf.String())
			}
		})
	}
	diffDeltas("source", c.Output.Source, c.Input.Source)
	diffDeltas("target", c.Output.Target, c.Input.Target)
	diffDeltas("head", c.Output.Head, c.Input.Head)
	diffDeltas("inclusion delay", c.Output.InclusionDelay, c.Input.InclusionDelay)
	diffDeltas("inactivity", c.Output.Inactivity, c.Input.Inactivity)
}

func (c *RewardsTest) Run() error {
	epc, err := c.Spec.NewEpochsContext(c.Pre)
	if err != nil {
		return err
	}
	process, err := c.Spec.PrepareEpochProcess(context.Background(), epc, c.Pre)
	if err != nil {
		return err
	}
	c.Output, err = c.Spec.AttestationRewardsAndPenalties(context.Background(), epc, process, c.Pre)
	return err
}

func TestAllDeltas(t *testing.T) {
	test_util.RunTransitionTest(t, "rewards", "core", func() test_util.TransitionTest {
		return &RewardsTest{}
	})
}
