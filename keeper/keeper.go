package keeper

import (
	"fmt"
	"path/filepath"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	storetypes "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/codec"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/janction/videoRendering"
	"github.com/janction/videoRendering/db"
)

type Keeper struct {
	cdc          codec.BinaryCodec
	addressCodec address.Codec
	BankKeeper   bankkeeper.BaseKeeper

	// authority is the address capable of executing a MsgUpdateParams and other authority-gated message.
	// typically, this should be the x/gov module account.
	authority string

	// ZKP
	ProvingKeyPath    string
	ValidatingKeyPath string

	// state management
	Schema                 collections.Schema
	Params                 collections.Item[videoRendering.Params]
	VideoRenderingTaskInfo collections.Item[videoRendering.VideoRenderingTaskInfo]
	VideoRenderingTasks    collections.Map[string, videoRendering.VideoRenderingTask]
	Workers                collections.Map[string, videoRendering.Worker]
	Configuration          VideoConfiguration
	DB                     db.DB
}

// NewKeeper creates a new Keeper instance
func NewKeeper(cdc codec.BinaryCodec, addressCodec address.Codec, storeService storetypes.KVStoreService, authority string, path string, bankKeeper bankkeeper.BaseKeeper) Keeper {
	if _, err := addressCodec.StringToBytes(authority); err != nil {
		panic(fmt.Errorf("invalid authority address: %w", err))
	}

	// we initialize the database
	db, err := db.Init(path)
	if err != nil {
		panic(err)
	}

	config, _ := GetVideoRenderingConfiguration(path)

	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		cdc:                    cdc,
		addressCodec:           addressCodec,
		authority:              authority,
		Params:                 collections.NewItem(sb, videoRendering.ParamsKey, "params", codec.CollValue[videoRendering.Params](cdc)),
		VideoRenderingTaskInfo: collections.NewItem(sb, videoRendering.TaskInfoKey, "taskInfo", codec.CollValue[videoRendering.VideoRenderingTaskInfo](cdc)),
		VideoRenderingTasks:    collections.NewMap(sb, videoRendering.VideoRenderingTaskKey, "videoRenderingTasks", collections.StringKey, codec.CollValue[videoRendering.VideoRenderingTask](cdc)),
		Workers:                collections.NewMap(sb, videoRendering.WorkerKey, "workers", collections.StringKey, codec.CollValue[videoRendering.Worker](cdc)),
		Configuration:          *config,
		DB:                     *db,
		BankKeeper:             bankKeeper,
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}

	// we set the paths of the ZKP proving and verifying keys
	k.ProvingKeyPath = filepath.Join(config.RootPath, "proving_key.pk")
	k.ValidatingKeyPath = filepath.Join(config.RootPath, "verifying_key.vk")

	k.Schema = schema

	return k
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}
