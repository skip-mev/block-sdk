/*
NOTE: These types are TEMPORARY and will be removed once the Cosmos SDK v0.48
alpha/RC tag is released. These types are simply used to prototype and develop
against.
*/
//nolint
package v2

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	ResponseVerifyVoteExtension_UNKNOWN ResponseVerifyVoteExtension_VerifyStatus = 0
	ResponseVerifyVoteExtension_ACCEPT  ResponseVerifyVoteExtension_VerifyStatus = 1
	// Rejecting the vote extension will reject the entire precommit by the sender.
	// Incorrectly implementing this thus has liveness implications as it may affect
	// CometBFT's ability to receive 2/3+ valid votes to finalize the block.
	// Honest nodes should never be rejected.
	ResponseVerifyVoteExtension_REJECT ResponseVerifyVoteExtension_VerifyStatus = 2
)

type (
	RequestExtendVote struct {
		// the hash of the block  that this vote may be referring to
		Hash []byte `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`
		// the height of the extended vote
		Height int64 `protobuf:"varint,2,opt,name=height,proto3" json:"height,omitempty"`
	}

	ResponseExtendVote struct {
		VoteExtension []byte `protobuf:"bytes,1,opt,name=vote_extension,json=voteExtension,proto3" json:"vote_extension,omitempty"`
	}

	RequestVerifyVoteExtension struct {
		// the hash of the block that this received vote corresponds to
		Hash []byte `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`
		// the validator that signed the vote extension
		ValidatorAddress []byte `protobuf:"bytes,2,opt,name=validator_address,json=validatorAddress,proto3" json:"validator_address,omitempty"`
		Height           int64  `protobuf:"varint,3,opt,name=height,proto3" json:"height,omitempty"`
		VoteExtension    []byte `protobuf:"bytes,4,opt,name=vote_extension,json=voteExtension,proto3" json:"vote_extension,omitempty"`
	}

	ResponseVerifyVoteExtension_VerifyStatus int32

	ResponseVerifyVoteExtension struct {
		Status ResponseVerifyVoteExtension_VerifyStatus `protobuf:"varint,1,opt,name=status,proto3,enum=tendermint.abci.ResponseVerifyVoteExtension_VerifyStatus" json:"status,omitempty"`
	}
)

type (
	// ExtendVoteHandler defines a function type alias for extending a pre-commit vote.
	ExtendVoteHandler func(sdk.Context, *RequestExtendVote) (*ResponseExtendVote, error)

	// VerifyVoteExtensionHandler defines a function type alias for verifying a
	// pre-commit vote extension.
	VerifyVoteExtensionHandler func(sdk.Context, *RequestVerifyVoteExtension) (*ResponseVerifyVoteExtension, error)
)
