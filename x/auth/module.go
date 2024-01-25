package auth

import (
	"context"
	"encoding/json"
	"fmt"

	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"

	"cosmossdk.io/core/address"
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/runtime/v2"
	"cosmossdk.io/server/v2/core/transaction"
	"cosmossdk.io/x/auth/ante"
	"cosmossdk.io/x/auth/keeper"
	"cosmossdk.io/x/auth/simulation"
	"cosmossdk.io/x/auth/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
)

// ConsensusVersion defines the current x/auth module consensus version.
const (
	ConsensusVersion = 5
	GovModuleName    = "gov"
)

var (
	_ module.AppModuleBasic      = AppModule{}
	_ module.AppModuleSimulation = AppModule{}
	_ module.HasGenesis          = AppModule{}
	_ module.HasServices         = AppModule{}

	_ appmodule.AppModule                       = AppModule{}
	_ appmodule.HasTxValidation[transaction.Tx] = AppModule{}
)

// AppModuleBasic defines the basic application module used by the auth module.
type AppModuleBasic struct {
	ac address.Codec
}

// Name returns the auth module's name.
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// RegisterLegacyAminoCodec registers the auth module's types for the given codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

// DefaultGenesis returns default genesis state as raw bytes for the auth
// module.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

// ValidateGenesis performs genesis state validation for the auth module.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	var data types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &data); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}

	return types.ValidateGenesis(data)
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the auth module.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *gwruntime.ServeMux) {
	if err := types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

// RegisterInterfaces registers interfaces and implementations of the auth module.
func (AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(registry)
}

// AppModule implements an application module for the auth module.
type AppModule struct {
	AppModuleBasic

	accountKeeper     keeper.AccountKeeper
	randGenAccountsFn types.RandomGenesisAccountsFn
}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}

// NewAppModule creates a new AppModule object
func NewAppModule(cdc codec.Codec, accountKeeper keeper.AccountKeeper, randGenAccountsFn types.RandomGenesisAccountsFn) AppModule {
	return AppModule{
		AppModuleBasic:    AppModuleBasic{ac: accountKeeper.AddressCodec()},
		accountKeeper:     accountKeeper,
		randGenAccountsFn: randGenAccountsFn,
	}
}

// RegisterServices registers a GRPC query service to respond to the
// module-specific GRPC queries.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.accountKeeper))
	types.RegisterQueryServer(cfg.QueryServer(), keeper.NewQueryServer(am.accountKeeper))

	m := keeper.NewMigrator(am.accountKeeper, cfg.QueryServer())
	if err := cfg.RegisterMigration(types.ModuleName, 1, m.Migrate1to2); err != nil {
		panic(fmt.Sprintf("failed to migrate x/%s from version 1 to 2: %v", types.ModuleName, err))
	}

	if err := cfg.RegisterMigration(types.ModuleName, 2, m.Migrate2to3); err != nil {
		panic(fmt.Sprintf("failed to migrate x/%s from version 2 to 3: %v", types.ModuleName, err))
	}

	if err := cfg.RegisterMigration(types.ModuleName, 3, m.Migrate3to4); err != nil {
		panic(fmt.Sprintf("failed to migrate x/%s from version 3 to 4: %v", types.ModuleName, err))
	}
	if err := cfg.RegisterMigration(types.ModuleName, 4, m.Migrate4To5); err != nil {
		panic(fmt.Sprintf("failed to migrate x/%s from version 4 to 5", types.ModuleName))
	}
}

// InitGenesis performs genesis initialization for the auth module. It returns
// no validator updates.
func (am AppModule) InitGenesis(ctx context.Context, cdc codec.JSONCodec, data json.RawMessage) {
	var genesisState types.GenesisState
	cdc.MustUnmarshalJSON(data, &genesisState)
	am.accountKeeper.InitGenesis(ctx, genesisState)
}

// ExportGenesis returns the exported genesis state as raw bytes for the auth
// module.
func (am AppModule) ExportGenesis(ctx context.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := am.accountKeeper.ExportGenesis(ctx)
	return cdc.MustMarshalJSON(gs)
}

// TxValidator implements appmodule.HasTxValidation.
// It replaces auth ante handlers for server/v2
func (am AppModule) TxValidator(ctx context.Context, tx transaction.Tx) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// supports legacy ante handler
	// eventually do the reverse, write ante handler as TxValidator
	anteDecorators := []sdk.AnteDecorator{
		ante.NewSetUpContextDecorator(),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(am.accountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(am.accountKeeper),
		ante.NewSetPubKeyDecorator(am.accountKeeper),
		ante.NewValidateSigCountDecorator(am.accountKeeper),
	}

	anteHandler := sdk.ChainAnteDecorators(anteDecorators...)
	_, err := anteHandler(sdkCtx, runtime.ServerTxToSDKTx(tx), sdkCtx.ExecMode() == sdk.ExecModeSimulate)
	return err
}

// ConsensusVersion implements AppModule/ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return ConsensusVersion }

// AppModuleSimulation functions

// GenerateGenesisState creates a randomized GenState of the auth module
func (am AppModule) GenerateGenesisState(simState *module.SimulationState) {
	simulation.RandomizedGenState(simState, am.randGenAccountsFn)
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return simulation.ProposalMsgs()
}

// RegisterStoreDecoder registers a decoder for auth module's types
func (am AppModule) RegisterStoreDecoder(sdr simtypes.StoreDecoderRegistry) {
	sdr[types.StoreKey] = simtypes.NewStoreDecoderFuncFromCollectionsSchema(am.accountKeeper.Schema)
}

// WeightedOperations doesn't return any auth module operation.
func (AppModule) WeightedOperations(_ module.SimulationState) []simtypes.WeightedOperation {
	return nil
}
