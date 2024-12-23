package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	storetypes "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/janction/videoRendering"
)

type Keeper struct {
	cdc          codec.BinaryCodec
	addressCodec address.Codec

	// authority is the address capable of executing a MsgUpdateParams and other authority-gated message.
	// typically, this should be the x/gov module account.
	authority string

	// state management
	Schema collections.Schema
	Params collections.Item[videoRendering.Params]

	VideoRenderingTasks collections.Map[string, videoRendering.VideoRenderingTask]
}

// NewKeeper creates a new Keeper instance
func NewKeeper(cdc codec.BinaryCodec, addressCodec address.Codec, storeService storetypes.KVStoreService, authority string) Keeper {
	if _, err := addressCodec.StringToBytes(authority); err != nil {
		panic(fmt.Errorf("invalid authority address: %w", err))
	}

	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		cdc:                 cdc,
		addressCodec:        addressCodec,
		authority:           authority,
		Params:              collections.NewItem(sb, videoRendering.ParamsKey, "params", codec.CollValue[videoRendering.Params](cdc)),
		VideoRenderingTasks: collections.NewMap(sb, videoRendering.VideoRenderingTaskKey, "videoRenderingTasks", collections.StringKey, codec.CollValue[videoRendering.VideoRenderingTask](cdc)),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}

	k.Schema = schema

	return k
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}
