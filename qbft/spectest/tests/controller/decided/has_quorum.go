package decided

import (
	"github.com/bloxapp/ssv-spec/qbft"
	"github.com/bloxapp/ssv-spec/qbft/spectest/tests"
	"github.com/bloxapp/ssv-spec/types"
	"github.com/bloxapp/ssv-spec/types/testingutils"
	"github.com/herumi/bls-eth-go-binary/bls"
)

// HasQuorum tests decided msg with unique 2f+1 signers
func HasQuorum() tests.SpecTest {
	ks := testingutils.Testing4SharesSet()
	return &tests.ControllerSpecTest{
		Name: "decide has quorum",
		RunInstanceData: []*tests.RunInstanceData{
			{
				InputValue: []byte{1, 2, 3, 4},
				InputMessages: []*qbft.SignedMessage{
					testingutils.TestingCommitMultiSignerMessageWithHeight([]*bls.SecretKey{ks.Shares[1], ks.Shares[2], ks.Shares[3]}, []types.OperatorID{1, 2, 3}, 10),
				},
				ExpectedDecidedState: tests.DecidedState{
					DecidedCnt:               1,
					DecidedVal:               testingutils.TestingQBFTFullData,
					CalledSyncDecidedByRange: true,
					DecidedByRangeValues:     [2]qbft.Height{0, 10},
				},
				ControllerPostRoot: "589b0c0352f1c22875246f2e66530d5fda62f646434b250ade128c61c16f47bd",
			},
		},
	}
}
