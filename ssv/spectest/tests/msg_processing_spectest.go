package tests

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	typescomparable "github.com/bloxapp/ssv-spec/types/testingutils/comparable"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bloxapp/ssv-spec/ssv"
	"github.com/bloxapp/ssv-spec/types"
	"github.com/bloxapp/ssv-spec/types/testingutils"
)

type MsgProcessingSpecTest struct {
	Name                    string
	Runner                  ssv.Runner
	Duty                    *types.Duty
	Messages                []*types.SSVMessage
	PostDutyRunnerStateRoot string
	PostDutyRunnerState     types.Root `json:"-"` // Field is ignored by encoding/json
	// OutputMessages compares pre/ post signed partial sigs to output. We exclude consensus msgs as it's tested in consensus
	OutputMessages         []*types.SignedPartialSignatureMessage
	BeaconBroadcastedRoots []string
	DontStartDuty          bool // if set to true will not start a duty for the runner
	ExpectedError          string
}

func (test *MsgProcessingSpecTest) TestName() string {
	return test.Name
}

// RunAsPartOfMultiTest runs the test as part of a MultiMsgProcessingSpecTest
func (test *MsgProcessingSpecTest) RunAsPartOfMultiTest(t *testing.T) {
	v, lastErr := test.runPreTesting()

	if len(test.ExpectedError) != 0 {
		require.EqualError(t, lastErr, test.ExpectedError)
	} else {
		require.NoError(t, lastErr)
	}

	// test output message
	test.compareOutputMsgs(t, v)

	// test beacon broadcasted msgs
	test.compareBroadcastedBeaconMsgs(t)

	// post root
	postRoot, err := test.Runner.GetRoot()
	require.NoError(t, err)

	if test.PostDutyRunnerStateRoot != hex.EncodeToString(postRoot[:]) {
		diff := typescomparable.PrintDiff(test.Runner, test.PostDutyRunnerState)
		require.EqualValues(t, test.PostDutyRunnerStateRoot, hex.EncodeToString(postRoot[:]), fmt.Sprintf("post runner state not equal\n%s\n", diff))
	}
}

// Run as an individual test
func (test *MsgProcessingSpecTest) Run(t *testing.T) {
	test.overrideStateComparison(t)
	test.RunAsPartOfMultiTest(t)
}

func (test *MsgProcessingSpecTest) runPreTesting() (*ssv.Validator, error) {
	v := testingutils.BaseValidator(testingutils.KeySetForShare(test.Runner.GetBaseRunner().Share))
	v.DutyRunners[test.Runner.GetBaseRunner().BeaconRoleType] = test.Runner
	v.Network = test.Runner.GetNetwork()

	var lastErr error
	if !test.DontStartDuty {
		lastErr = v.StartDuty(test.Duty)
	}
	for _, msg := range test.Messages {
		err := v.ProcessMessage(msg)
		if err != nil {
			lastErr = err
		}
	}

	return v, lastErr
}

func (test *MsgProcessingSpecTest) compareBroadcastedBeaconMsgs(t *testing.T) {
	broadcastedRoots := test.Runner.GetBeaconNode().(*testingutils.TestingBeaconNode).BroadcastedRoots
	require.Len(t, broadcastedRoots, len(test.BeaconBroadcastedRoots))
	for _, r1 := range test.BeaconBroadcastedRoots {
		found := false
		for _, r2 := range broadcastedRoots {
			if r1 == hex.EncodeToString(r2[:]) {
				found = true
				break
			}
		}
		require.Truef(t, found, "broadcasted beacon root not found")
	}
}

func (test *MsgProcessingSpecTest) compareOutputMsgs(t *testing.T, v *ssv.Validator) {
	filterPartialSigs := func(messages []*types.SSVMessage) []*types.SSVMessage {
		ret := make([]*types.SSVMessage, 0)
		for _, msg := range messages {
			if msg.MsgType != types.SSVPartialSignatureMsgType {
				continue
			}
			ret = append(ret, msg)
		}
		return ret
	}
	broadcastedMsgs := filterPartialSigs(v.Network.(*testingutils.TestingNetwork).BroadcastedMsgs)
	require.Len(t, broadcastedMsgs, len(test.OutputMessages))
	index := 0
	for _, msg := range broadcastedMsgs {
		if msg.MsgType != types.SSVPartialSignatureMsgType {
			continue
		}

		msg1 := &types.SignedPartialSignatureMessage{}
		require.NoError(t, msg1.Decode(msg.Data))
		msg2 := test.OutputMessages[index]
		require.Len(t, msg1.Message.Messages, len(msg2.Message.Messages))

		// messages are not guaranteed to be in order so we map them and then test all roots to be equal
		roots := make(map[string]string)
		for i, partialSigMsg2 := range msg2.Message.Messages {
			r2, err := partialSigMsg2.GetRoot()
			require.NoError(t, err)
			if _, found := roots[hex.EncodeToString(r2[:])]; !found {
				roots[hex.EncodeToString(r2[:])] = ""
			} else {
				roots[hex.EncodeToString(r2[:])] = hex.EncodeToString(r2[:])
			}

			partialSigMsg1 := msg1.Message.Messages[i]
			r1, err := partialSigMsg1.GetRoot()
			require.NoError(t, err)

			if _, found := roots[hex.EncodeToString(r1[:])]; !found {
				roots[hex.EncodeToString(r1[:])] = ""
			} else {
				roots[hex.EncodeToString(r1[:])] = hex.EncodeToString(r1[:])
			}
		}
		for k, v := range roots {
			require.EqualValues(t, k, v, "missing output msg")
		}

		// test that slot is correct in broadcasted msg
		require.EqualValues(t, msg1.Message.Slot, msg2.Message.Slot, "incorrect broadcasted slot")

		index++
	}
}

func (test *MsgProcessingSpecTest) overrideStateComparison(t *testing.T) {
	basedir, _ := os.Getwd()
	path := filepath.Join(basedir, "generate", "state_comparison", reflect.TypeOf(test).String(), fmt.Sprintf("%s.json", test.TestName()))
	byteValue, err := os.ReadFile(path)
	require.NoError(t, err)

	switch test.Runner.(type) {
	case *ssv.AttesterRunner:
		test.PostDutyRunnerState = &ssv.AttesterRunner{}
	case *ssv.AggregatorRunner:
		test.PostDutyRunnerState = &ssv.AggregatorRunner{}
	case *ssv.ProposerRunner:
		test.PostDutyRunnerState = &ssv.ProposerRunner{}
	case *ssv.SyncCommitteeRunner:
		test.PostDutyRunnerState = &ssv.SyncCommitteeRunner{}
	case *ssv.SyncCommitteeAggregatorRunner:
		test.PostDutyRunnerState = &ssv.SyncCommitteeAggregatorRunner{}
	case *ssv.ValidatorRegistrationRunner:
		test.PostDutyRunnerState = &ssv.ValidatorRegistrationRunner{}
	default:
		t.Fatalf("unknown runner type")
	}
	require.NoError(t, json.Unmarshal(byteValue, &test.PostDutyRunnerState))

	r, err := test.PostDutyRunnerState.GetRoot()
	require.NoError(t, err)

	// backwards compatability test, hard coded post root must be equal to the one loaded from file
	if len(test.PostDutyRunnerStateRoot) > 0 {
		require.EqualValues(t, test.PostDutyRunnerStateRoot, hex.EncodeToString(r[:]))
	}

	test.PostDutyRunnerStateRoot = hex.EncodeToString(r[:])
}

func (test *MsgProcessingSpecTest) GetPostState() (interface{}, error) {
	_, lastErr := test.runPreTesting()
	if lastErr != nil && len(test.ExpectedError) == 0 {
		return nil, lastErr
	}

	return test.Runner, nil
}
