package operations

import (
	"github.com/protolambda/zrnt/eth2/beacon"
	"github.com/protolambda/zrnt/tests/spec/test_util"
	"testing"
)

type ProposerSlashingTestCase struct {
	test_util.BaseTransitionTest
	ProposerSlashing beacon.ProposerSlashing
}

func (c *ProposerSlashingTestCase) Load(t *testing.T, readPart test_util.TestPartReader) {
	c.BaseTransitionTest.Load(t, readPart)
	test_util.LoadSSZ(t, "proposer_slashing", &c.ProposerSlashing, readPart)
}

func (c *ProposerSlashingTestCase) Run() error {
	epc, err := c.Spec.NewEpochsContext(c.Pre)
	if err != nil {
		return err
	}
	return c.Spec.ProcessProposerSlashing(epc, c.Pre, &c.ProposerSlashing)
}

func TestProposerSlashing(t *testing.T) {
	test_util.RunTransitionTest(t, "operations", "proposer_slashing",
		func() test_util.TransitionTest { return new(ProposerSlashingTestCase) })
}
