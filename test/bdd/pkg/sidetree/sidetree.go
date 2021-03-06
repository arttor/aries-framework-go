/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sidetree

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/btcsuite/btcutil/base58"
	"github.com/trustbloc/sidetree-core-go/pkg/commitment"
	"github.com/trustbloc/sidetree-core-go/pkg/document"
	"github.com/trustbloc/sidetree-core-go/pkg/restapi/helper"
	"github.com/trustbloc/sidetree-core-go/pkg/util/pubkey"

	diddoc "github.com/hyperledger/aries-framework-go/pkg/doc/did"
	"github.com/hyperledger/aries-framework-go/pkg/doc/jose"
	"github.com/hyperledger/aries-framework-go/test/bdd/pkg/util"
)

const docTemplate = `{
  "publicKey": [
   {
     "id": "%s",
     "type": "%s",
     "purpose": ["auth", "general"],
     "jwk": %s
   }
  ],
  "service": [
	{
	   "id": "hub",
	   "type": "did-communication",
	   "endpoint": "%s",
       "recipientKeys" : [ "%s" ]
	}
  ]
}`

const (
	sha2_256       = 18
	defaultKeyType = "JwsVerificationKey2020"
)

type didResolution struct {
	Context          interface{}     `json:"@context"`
	DIDDocument      json.RawMessage `json:"didDocument"`
	ResolverMetadata json.RawMessage `json:"resolverMetadata"`
	MethodMetadata   json.RawMessage `json:"methodMetadata"`
}

// CreateDIDParams defines parameters for CreateDID().
type CreateDIDParams struct {
	URL             string
	KeyID           string
	JWK             *jose.JWK
	KeyType         string
	ServiceEndpoint string
}

// CreateDID in sidetree
func CreateDID(params *CreateDIDParams) (*diddoc.Doc, error) {
	opaqueDoc, err := getOpaqueDocument(params)
	if err != nil {
		return nil, err
	}

	req, err := getCreateRequest(opaqueDoc, params.JWK)
	if err != nil {
		return nil, err
	}

	var result didResolution

	err = util.SendHTTP(http.MethodPost, params.URL, req, &result)
	if err != nil {
		return nil, err
	}

	doc, err := diddoc.ParseDocument(result.DIDDocument)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public DID document: %s", err)
	}

	return doc, nil
}

func getOpaqueDocument(params *CreateDIDParams) ([]byte, error) {
	opsPubKey, err := getPubKey(params.JWK)
	if err != nil {
		return nil, err
	}

	keyBytes, err := params.JWK.PublicKeyBytes()
	if err != nil {
		return nil, err
	}

	keyType := params.KeyType
	if keyType == "" {
		keyType = defaultKeyType
	}

	data := fmt.Sprintf(docTemplate, params.KeyID, keyType, opsPubKey, params.ServiceEndpoint, base58.Encode(keyBytes))

	doc, err := document.FromBytes([]byte(data))
	if err != nil {
		return nil, err
	}

	return doc.Bytes()
}

func getPubKey(jwk *jose.JWK) (string, error) {
	publicKey, err := pubkey.GetPublicKeyJWK(jwk.Key)
	if err != nil {
		return "", err
	}

	opsPubKeyBytes, err := json.Marshal(publicKey)
	if err != nil {
		return "", err
	}

	return string(opsPubKeyBytes), nil
}

func getCreateRequest(doc []byte, jwk *jose.JWK) ([]byte, error) {
	pubKey, err := pubkey.GetPublicKeyJWK(jwk.Key)
	if err != nil {
		return nil, err
	}

	c, err := commitment.Calculate(pubKey, sha2_256)
	if err != nil {
		return nil, err
	}

	// for testing purposes we are going to use same commitment key for update and recovery
	return helper.NewCreateRequest(&helper.CreateRequestInfo{
		OpaqueDocument:     string(doc),
		UpdateCommitment:   c,
		RecoveryCommitment: c,
		MultihashCode:      sha2_256,
	})
}
