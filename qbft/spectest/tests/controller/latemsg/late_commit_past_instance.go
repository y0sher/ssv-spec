package latemsg

import (
	"github.com/herumi/bls-eth-go-binary/bls"

	"github.com/bloxapp/ssv-spec/qbft"
	"github.com/bloxapp/ssv-spec/qbft/spectest/tests"
	"github.com/bloxapp/ssv-spec/types"
	"github.com/bloxapp/ssv-spec/types/testingutils"
	"github.com/bloxapp/ssv-spec/types/testingutils/comparable"
)

// LateCommitPastInstance tests process commit msg for a previously decided instance
func LateCommitPastInstance() tests.SpecTest {
	ks := testingutils.Testing4SharesSet()

	allMsgs := testingutils.DecidingMsgsForHeightWithRoot(testingutils.TestingQBFTRootData,
		testingutils.TestingQBFTFullData, testingutils.TestingIdentifier, 1, ks)

	msgPerHeight := make(map[qbft.Height][]*qbft.SignedMessage)
	msgPerHeight[qbft.FirstHeight] = allMsgs[0:7]
	msgPerHeight[1] = allMsgs[7:14]

	instanceData := func(height qbft.Height) *tests.RunInstanceData {
		sc := lateCommitPastInstanceStateComparison(height, nil)
		return &tests.RunInstanceData{
			InputValue:    []byte{1, 2, 3, 4},
			InputMessages: msgPerHeight[height],
			ExpectedDecidedState: tests.DecidedState{
				BroadcastedDecided: testingutils.TestingCommitMultiSignerMessageWithHeight(
					[]*bls.SecretKey{ks.Shares[1], ks.Shares[2], ks.Shares[3]},
					[]types.OperatorID{1, 2, 3},
					height,
				),
				DecidedVal: testingutils.TestingQBFTFullData,
				DecidedCnt: 1,
			},
			ControllerPostRoot:  sc.Root(),
			ControllerPostState: sc.ExpectedState,
		}
	}

	lateMsg := testingutils.TestingCommitMultiSignerMessageWithHeight([]*bls.SecretKey{ks.Shares[4]}, []types.OperatorID{4}, qbft.FirstHeight)
	sc := lateCommitPastInstanceStateComparison(2, lateMsg)

	return &tests.ControllerSpecTest{
		Name: "late commit past instance",
		RunInstanceData: []*tests.RunInstanceData{
			instanceData(qbft.FirstHeight),
			instanceData(1),
			{
				InputValue: []byte{1, 2, 3, 4},
				InputMessages: []*qbft.SignedMessage{
					lateMsg,
				},
				ControllerPostRoot:  sc.Root(),
				ControllerPostState: sc.ExpectedState,
			},
		},
	}
}

// lateCommitPastInstanceStateComparison returns a comparable.StateComparison for controller running up until the given height.
// lateMsg will be added to the commit container of the instance at the proper height.
func lateCommitPastInstanceStateComparison(height qbft.Height, lateMsg *qbft.SignedMessage) *comparable.StateComparison {
	ks := testingutils.Testing4SharesSet()
	allMsgs := testingutils.ExpectedDecidingMsgsForHeightWithRoot(testingutils.TestingQBFTRootData, testingutils.TestingQBFTFullData, testingutils.TestingIdentifier, 1, ks)
	offset := 7 // 7 messages per height (1 propose + 3 prepare + 3 commit)

	contr := testingutils.NewTestingQBFTController(
		testingutils.TestingIdentifier,
		testingutils.TestingShare(testingutils.Testing4SharesSet()),
		testingutils.TestingConfig(testingutils.Testing4SharesSet()),
	)

	for i := 0; i <= int(height); i++ {
		contr.Height = qbft.Height(i)

		instance := &qbft.Instance{
			StartValue: []byte{1, 2, 3, 4},
			State: &qbft.State{
				Share:  testingutils.TestingShare(testingutils.Testing4SharesSet()),
				ID:     testingutils.TestingIdentifier,
				Round:  qbft.FirstRound,
				Height: qbft.Height(i),
			},
		}

		// last height should be just initialized, since no messages were processed for it
		if lateMsg != nil && qbft.Height(i) == height {
			comparable.InitContainers(instance)
			contr.StoredInstances = append([]*qbft.Instance{instance}, contr.StoredInstances...)
			break
		}

		instance.State.ProposalAcceptedForCurrentRound = testingutils.TestingProposalMessageWithParams(ks.Shares[1], types.OperatorID(1), qbft.FirstRound, qbft.Height(i), testingutils.TestingQBFTRootData, nil, nil)
		instance.State.LastPreparedRound = qbft.FirstRound
		instance.State.LastPreparedValue = testingutils.TestingQBFTFullData
		instance.State.Decided = true
		instance.State.DecidedValue = testingutils.TestingQBFTFullData

		msgs := allMsgs[offset*i : offset*(i+1)]
		comparable.SetSignedMessages(instance, msgs)

		if lateMsg != nil && qbft.Height(i) == lateMsg.Message.Height {
			instance.State.CommitContainer.Msgs[qbft.FirstRound] = append(instance.State.CommitContainer.Msgs[qbft.FirstRound], lateMsg)
		}

		contr.StoredInstances = append([]*qbft.Instance{instance}, contr.StoredInstances...)
	}

	return &comparable.StateComparison{ExpectedState: contr}
}
