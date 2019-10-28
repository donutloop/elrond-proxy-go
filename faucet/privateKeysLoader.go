package faucet

import (
	"encoding/hex"
	"errors"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/crypto/signing"
	"github.com/ElrondNetwork/elrond-go/crypto/signing/kyber"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"strings"
)

func getSuite() crypto.Suite {
	return kyber.NewBlakeSHA256Ed25519()
}

type PrivateKeysLoader struct {
	addrConv        state.AddressConverter
	keyGen          crypto.KeyGenerator
	pemFileLocation string
	shardCoord      sharding.Coordinator
}

func NewPrivateKeysLoader(
	addrConv state.AddressConverter,
	shardCoord sharding.Coordinator,
	pemFileLocation string,
) (*PrivateKeysLoader, error) {
	if addrConv == nil {
		return nil, errors.New("nil address converter")
	}
	if shardCoord == nil {
		return nil, errors.New("nil shard coordinator")
	}
	if len(pemFileLocation) == 0 {
		return nil, errors.New("invalid pem file location")
	}

	keyGen := signing.NewKeyGenerator(getSuite())
	return &PrivateKeysLoader{
		addrConv:        addrConv,
		keyGen:          keyGen,
		shardCoord:      shardCoord,
		pemFileLocation: pemFileLocation,
	}, nil
}

func (pkl *PrivateKeysLoader) MapOfPrivateKeysByShard() (map[uint32][]crypto.PrivateKey, error) {
	privKeysMapByShard := make(map[uint32][]crypto.PrivateKey)
	privKeysBytes, err := pkl.loadPrivKeysBytesFromPemFile()
	if err != nil {
		return nil, err
	}

	for _, privKeyBytes := range privKeysBytes {
		privKeyBytes, err := hex.DecodeString(string(privKeyBytes))
		if err != nil {
			return nil, err
		}

		privKey, err := pkl.keyGen.PrivateKeyFromByteArray(privKeyBytes)
		if err != nil {
			return nil, err
		}

		pubKeyOfPrivKey, err := pkl.pubKeyFromPrivKey(privKey)
		if err != nil {
			return nil, err
		}

		address, err := pkl.addrConv.CreateAddressFromPublicKeyBytes(pubKeyOfPrivKey)
		if err != nil {
			return nil, err
		}

		shardId := pkl.shardCoord.ComputeId(address)

		privKeysMapByShard[shardId] = append(privKeysMapByShard[shardId], privKey)
	}

	return privKeysMapByShard, nil
}

func (pkl *PrivateKeysLoader) loadPrivKeysBytesFromPemFile() ([][]byte, error) {
	var privateKeysSlice [][]byte
	index := 0
	for {
		sk, err := core.LoadSkFromPemFile(pkl.pemFileLocation, nil, index)
		if err != nil && strings.Contains(err.Error(), "invalid private key index") {
			if len(privateKeysSlice) == 0 {
				return nil, err
			}

			return privateKeysSlice, nil
		}
		privateKeysSlice = append(privateKeysSlice, sk)
		index++
	}
}

func (pkl *PrivateKeysLoader) pubKeyFromPrivKey(sk crypto.PrivateKey) ([]byte, error) {
	pk := sk.GeneratePublic()
	return pk.ToByteArray()
}