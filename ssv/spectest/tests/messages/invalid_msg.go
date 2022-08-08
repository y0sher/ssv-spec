package messages

import (
	"github.com/bloxapp/ssv-spec/qbft"
	"github.com/bloxapp/ssv-spec/ssv"
	"github.com/bloxapp/ssv-spec/types/testingutils"
)

// InvalidMsg tests a signed msg with 1 invalid message
func InvalidMsg() *MsgSpecTest {
	ks := testingutils.Testing4SharesSet()

	msg := testingutils.PostConsensusAttestationMsg(ks.Shares[1], 1, qbft.FirstHeight)
	msg.Messages = append(msg.Messages, &ssv.PartialSignatureMessage{})

	return &MsgSpecTest{
		Name:          "no messages",
		Messages:      []*ssv.SignedPartialSignatureMessage{msg},
		ExpectedError: "message invalid: PartialSignatureMessage sig invalid",
	}
}