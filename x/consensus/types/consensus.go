package types

import (
	"errors"

	cmtproto "github.com/cometbft/cometbft/api/cometbft/types/v1"
	cmttypes "github.com/cometbft/cometbft/types"
	gogotypes "github.com/cosmos/gogoproto/types"
)

func (msg ConsensusMsgParams) ToProtoConsensusParams() (cmtproto.ConsensusParams, error) {
	if msg.Evidence == nil || msg.Block == nil || msg.Validator == nil || msg.Version == nil {
		return cmtproto.ConsensusParams{}, errors.New("all parameters must be present")
	}

	cp := cmtproto.ConsensusParams{
		Block: &cmtproto.BlockParams{
			MaxBytes: msg.Block.MaxBytes,
			MaxGas:   msg.Block.MaxGas,
		},
		Evidence: &cmtproto.EvidenceParams{
			MaxAgeNumBlocks: msg.Evidence.MaxAgeNumBlocks,
			MaxAgeDuration:  msg.Evidence.MaxAgeDuration,
			MaxBytes:        msg.Evidence.MaxBytes,
		},
		Validator: &cmtproto.ValidatorParams{
			PubKeyTypes: msg.Validator.PubKeyTypes,
		},

		Version: cmttypes.DefaultConsensusParams().ToProto().Version, // Version is stored in x/upgrade
	}

	if msg.Abci != nil {
		cp.Feature = &cmtproto.FeatureParams{
			VoteExtensionsEnableHeight: &gogotypes.Int64Value{Value: msg.Abci.VoteExtensionsEnableHeight},
		}
	}

	if msg.Version != nil {
		cp.Version.App = msg.Version.App
	}

	return cp, nil
}
