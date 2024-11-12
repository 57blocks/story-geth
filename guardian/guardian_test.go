package guardian

import (
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/cipherowl-ai/addressdb/address"
	"github.com/cipherowl-ai/addressdb/store"
)

func TestInitInstance(t *testing.T) {
	type args struct {
		config Config
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "InstanceCreationWithValidConfig",
			args: args{
				config: Config{},
			},
			want: true,
		},
		{
			name: "InstanceCreationWithDisabledConfig",
			args: args{
				config: Config{
					Disabled: true,
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			InitInstance(tt.args.config)
		})
	}
}

func TestGuardian_CheckTransaction(t *testing.T) {
	type args struct {
		signer          types.Signer
		tx              *types.Transaction
		testFromAddress string
	}
	tests := []struct {
		name    string
		args    *args
		prepare func(a *args)
		want    bool
	}{
		{
			name: "not filtered",
			args: new(args),
			prepare: func(a *args) {
				key, _ := crypto.GenerateKey()
				signer := types.NewEIP155Signer(big.NewInt(18))

				tx, err := types.SignTx(types.NewTransaction(0, common.HexToAddress("0x810205E412eB4b9f8A7faEF8faE4cF08D7c680e1"), new(big.Int), 0, new(big.Int), nil), signer, key)
				if err != nil {
					t.Fatal(err)
				}

				a.signer = signer
				a.tx = tx
			},
			want: false,
		}, {
			name: "should filter 'to' address",
			args: new(args),
			prepare: func(a *args) {
				key, _ := crypto.GenerateKey()
				signer := types.NewEIP155Signer(big.NewInt(18))

				tx, err := types.SignTx(types.NewTransaction(0, common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"), new(big.Int), 0, new(big.Int), nil), signer, key)
				if err != nil {
					t.Fatal(err)
				}

				a.signer = signer
				a.tx = tx
			},
			want: true,
		},
		{
			name: "should filter 'from' address",
			args: new(args),
			prepare: func(a *args) {
				key, _ := crypto.GenerateKey()
				signer := types.NewEIP155Signer(big.NewInt(18))

				tx, err := types.SignTx(types.NewTransaction(0, common.HexToAddress("0x810205E412eB4b9f8A7faEF8faE4cF08D7c680e1"), new(big.Int), 0, new(big.Int), nil), signer, key)
				if err != nil {
					t.Fatal(err)
				}

				a.signer = signer
				a.tx = tx

				from, err := types.Sender(signer, tx)
				if err != nil {
					t.Fatal(err)
				}
				a.testFromAddress = from.Hex()
			},
			want: true,
		},
	}
	for _, tt := range tests {
		if tt.prepare != nil {
			tt.prepare(tt.args)
		}
		t.Run(tt.name, func(t *testing.T) {
			bf, err := store.NewBloomFilterStore(&address.EVMAddressHandler{})
			if err != nil {
				t.Fatal(err)
			}

			_ = bf.AddAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
			_ = bf.AddAddress("0x97DCA899a2278d010d678d64fBC7C718eD5D4939")
			if tt.args.testFromAddress != "" {
				_ = bf.AddAddress(tt.args.testFromAddress)
			}

			filterFilePath := saveBloomFilterToFile(t, bf)
			defer os.Remove(filterFilePath)

			InitInstance(Config{FilterFilePath: filterFilePath})
			g, err := GetInstance()
			if err != nil {
				t.Fatal(err)
			}
			defer instance.Reset()

			if got := g.CheckTransaction(tt.args.signer, tt.args.tx); got != tt.want {
				t.Errorf("CheckSanctionedTransaction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func saveBloomFilterToFile(t *testing.T, bf *store.BloomFilterStore) string {
	filePath := os.TempDir() + "/bloom_filter.gob"
	if err := bf.SaveToFile(filePath); err != nil {
		t.Fatalf("Failed to save Bloom filter to file: %v", err)
	}
	return filePath
}
