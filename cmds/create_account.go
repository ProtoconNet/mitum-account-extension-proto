package cmds

import (
	"bytes"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"

	"github.com/spikeekips/mitum-currency/currency"
)

type CreateAccountCommand struct {
	*BaseCommand
	Privatekey PrivatekeyFlag `arg:"" name:"privatekey" help:"sender's privatekey" required:""`
	Sender     AddressFlag    `arg:"" name:"sender" help:"sender address" required:""`
	Amount     AmountFlag     `arg:"" name:"amount" help:"amount to send" required:""`
	Threshold  uint           `help:"threshold for keys (default: ${create_account_threshold})" default:"${create_account_threshold}"` // nolint
	Token      string         `help:"token for operation" optional:""`
	NetworkID  NetworkIDFlag  `name:"network-id" help:"network-id" required:""`
	Keys       []KeyFlag      `name:"key" help:"key for new account (ex: \"<public key>,<weight>\")" sep:"@"`
	Pretty     bool           `name:"pretty" help:"pretty format"`
	Memo       string         `name:"memo" help:"memo"`
	Seal       FileLoad       `help:"seal" optional:""`
	sender     base.Address
	keys       currency.Keys
}

func NewCreateAccountCommand() CreateAccountCommand {
	return CreateAccountCommand{
		BaseCommand: NewBaseCommand("create-account-operation"),
	}
}

func (cmd *CreateAccountCommand) Run(version util.Version) error { // nolint:dupl
	if err := cmd.Initialize(cmd, version); err != nil {
		return xerrors.Errorf("failed to initialize command: %w", err)
	}

	if err := cmd.parseFlags(); err != nil {
		return err
	} else if a, err := cmd.Sender.Encode(jenc); err != nil {
		return xerrors.Errorf("invalid sender format, %q: %w", cmd.Sender.String(), err)
	} else {
		cmd.sender = a
	}

	var op operation.Operation
	if o, err := cmd.createOperation(); err != nil {
		return err
	} else {
		op = o
	}

	if sl, err := loadSealAndAddOperation(
		cmd.Seal.Bytes(),
		cmd.Privatekey,
		cmd.NetworkID.Bytes(),
		op,
	); err != nil {
		return err
	} else {
		cmd.pretty(cmd.Pretty, sl)
	}

	return nil
}

func (cmd *CreateAccountCommand) parseFlags() error {
	if len(cmd.Keys) < 1 {
		return xerrors.Errorf("--key must be given at least one")
	}

	if len(cmd.Token) < 1 {
		cmd.Token = localtime.String(localtime.Now())
	}

	{
		ks := make([]currency.Key, len(cmd.Keys))
		for i := range cmd.Keys {
			ks[i] = cmd.Keys[i].Key
		}

		if kys, err := currency.NewKeys(ks, cmd.Threshold); err != nil {
			return err
		} else if err := kys.IsValid(nil); err != nil {
			return err
		} else {
			cmd.keys = kys
		}
	}

	return nil
}

func (cmd *CreateAccountCommand) createOperation() (operation.Operation, error) {
	var items []currency.CreateAccountItem
	if len(bytes.TrimSpace(cmd.Seal.Bytes())) > 0 {
		var sl seal.Seal
		if s, err := loadSeal(cmd.Seal.Bytes(), cmd.NetworkID.Bytes()); err != nil {
			return nil, err
		} else if so, ok := s.(operation.Seal); !ok {
			return nil, xerrors.Errorf("seal is not operation.Seal, %T", s)
		} else if _, ok := so.(operation.SealUpdater); !ok {
			return nil, xerrors.Errorf("seal is not operation.SealUpdater, %T", s)
		} else {
			sl = so
		}

		for _, op := range sl.(operation.Seal).Operations() {
			if t, ok := op.(currency.CreateAccounts); ok {
				items = t.Fact().(currency.CreateAccountsFact).Items()
			}
		}
	}

	item := currency.NewCreateAccountItem(cmd.keys, cmd.Amount.Amount)
	if err := item.IsValid(nil); err != nil {
		return nil, err
	} else {
		items = append(items, item)
	}

	fact := currency.NewCreateAccountsFact([]byte(cmd.Token), cmd.sender, items)

	var fs []operation.FactSign
	if sig, err := operation.NewFactSignature(cmd.Privatekey, fact, []byte(cmd.NetworkID)); err != nil {
		return nil, err
	} else {
		fs = append(fs, operation.NewBaseFactSign(cmd.Privatekey.Publickey(), sig))
	}

	if op, err := currency.NewCreateAccounts(fact, fs, cmd.Memo); err != nil {
		return nil, xerrors.Errorf("failed to create create-account operation: %w", err)
	} else {
		return op, nil
	}
}

func loadSeal(b []byte, networkID base.NetworkID) (seal.Seal, error) {
	if len(bytes.TrimSpace(b)) < 1 {
		return nil, xerrors.Errorf("empty input")
	}

	if sl, err := seal.DecodeSeal(jenc, b); err != nil {
		return nil, err
	} else if err := sl.IsValid(networkID); err != nil {
		return nil, xerrors.Errorf("invalid seal: %w", err)
	} else {
		return sl, nil
	}
}

func loadSealAndAddOperation(
	b []byte,
	privatekey key.Privatekey,
	networkID base.NetworkID,
	op operation.Operation,
) (operation.Seal, error) {
	if b == nil {
		if bs, err := operation.NewBaseSeal(
			privatekey,
			[]operation.Operation{op},
			networkID,
		); err != nil {
			return nil, xerrors.Errorf("failed to create operation.Seal: %w", err)
		} else {
			return bs, nil
		}
	}

	var sl operation.Seal
	if s, err := loadSeal(b, networkID); err != nil {
		return nil, err
	} else if so, ok := s.(operation.Seal); !ok {
		return nil, xerrors.Errorf("seal is not operation.Seal, %T", s)
	} else if _, ok := so.(operation.SealUpdater); !ok {
		return nil, xerrors.Errorf("seal is not operation.SealUpdater, %T", s)
	} else {
		sl = so
	}

	// NOTE add operation to existing seal
	sl = sl.(operation.SealUpdater).SetOperations([]operation.Operation{op}).(operation.Seal)

	if s, err := signSeal(sl, privatekey, networkID); err != nil {
		return nil, err
	} else {
		sl = s.(operation.Seal)
	}

	return sl, nil
}
