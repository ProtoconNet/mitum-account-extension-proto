package extension

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
)

var (
	StateKeyContractAccountSuffix       = ":contractaccount"
	StateKeyContractAccountConfigSuffix = ":contractaccountconfig"
)

func StateKeyContractAccount(a base.Address) string {
	return fmt.Sprintf("%s%s", a.String(), StateKeyContractAccountSuffix)
}

func IsStateContractAccountKey(key string) bool {
	return strings.HasSuffix(key, StateKeyContractAccountSuffix)
}

func StateContractAccountValue(st state.State) (ContractAccount, error) {
	v := st.Value()
	if v == nil {
		return ContractAccount{}, util.NotFoundError.Errorf("contract account status not found in State")
	}

	s, ok := v.Interface().(ContractAccount)
	if !ok {
		return ContractAccount{}, errors.Errorf("invalid contract account status value found, %T", v.Interface())
	}
	return s, nil
}

func SetStateContractAccountValue(st state.State, v ContractAccount) (state.State, error) {
	uv, err := state.NewHintedValue(v)
	if err != nil {
		return nil, err
	}
	return st.SetValue(uv)
}

func StateKeyContractAccountConfig( /* model name */ m string, id string, a base.Address) string {
	return fmt.Sprintf("%s-%s-%s%s", m, id, a.String(), StateKeyContractAccountConfigSuffix)
}

func IsStateContractAccountConfigKey(key string) bool {
	return strings.HasSuffix(key, StateKeyContractAccountConfigSuffix)
}

func StateContractAccountConfigValue(st state.State) (Config, error) {
	v := st.Value()
	if v == nil {
		return nil, util.NotFoundError.Errorf("config not found in State")
	}

	s, ok := v.Interface().(Config)
	if !ok {
		return nil, errors.Errorf("invalid config value found, %T", v.Interface())
	}
	return s, nil
}

func setStateContractAccountConfigValue(st state.State, v Config) (state.State, error) {
	uv, err := state.NewHintedValue(v)
	if err != nil {
		return nil, err
	}
	return st.SetValue(uv)
}

func checkExistsState(
	key string,
	getState func(key string) (state.State, bool, error),
) error {
	switch _, found, err := getState(key); {
	case err != nil:
		return err
	case !found:
		return operation.NewBaseReasonError("state, %q does not exist", key)
	default:
		return nil
	}
}

func existsState(
	k,
	name string,
	getState func(key string) (state.State, bool, error),
) (state.State, error) {
	switch st, found, err := getState(k); {
	case err != nil:
		return nil, err
	case !found:
		return nil, operation.NewBaseReasonError("%s does not exist", name)
	default:
		return st, nil
	}
}

func notExistsState(
	k,
	name string,
	getState func(key string) (state.State, bool, error),
) (state.State, error) {
	switch st, found, err := getState(k); {
	case err != nil:
		return nil, err
	case found:
		return nil, operation.NewBaseReasonError("%s already exists", name)
	default:
		return st, nil
	}
}
