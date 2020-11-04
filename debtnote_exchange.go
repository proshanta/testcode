/*
 SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/json"
	"fmt"
	"log"

	"crypto/sha256"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

/*
// Debt Note struct and properties must be exported (start with capitals) to work with contract api metadata
type DebtNote struct {
	ObjectType        string `json:"objectType"` // ObjectType is used to distinguish different object types in the same chaincode namespace
	DebtNoteID        string `json:"debtNoteID"`
	DebtyOrg          string `json:"debtyOrg"`
	Debt	          int `json:"debt"`
	RedeemStatus	  string `json:"redeemStatus"`
}

// Debt Note struct and properties must be exported (start with capitals) to work with contract api metadata
type DebtNoteOwner struct {
	DebtNoteID        string `json:"debtNoteID"`
	NewOwnerOrg       string `json:"newOwnerOrg"`
}


//receipt of the debt note sharing
type DebtNoteReceipt struct {
	ObjectType        string
	DebtNoteID        string
	timestamp 		  time.Time
}


// for recipt composite key
const (
	typeCurrentOwnReceipt = "CO"
	typeNewOwnReceipt  = "NO"
)
*/
type SmartContract struct {
	contractapi.Contract
}

// CreateDebtNote creates a debt note and sets it as owned by the client's org
func (s *SmartContract) CreateDebtNote(ctx contractapi.TransactionContextInterface) error {

	// get the private data from transient parameter
	transientMap, err := ctx.GetStub().GetTransient()
	if err != nil {
		return fmt.Errorf("error getting transient: %v", err)
	}

	// Asset properties must be retrieved from the transient field as they are private
	immutableDebtNoteJSON, ok := transientMap["debtnote_properties"]
	if !ok {
		return fmt.Errorf("debtnote_properties key not found in the transient map")
	}

	// fetch the debt details
	//var debtNoteInput DebtNote

	var debtNoteInput map[string]interface{}
	debtNoteInput = make(map[string]interface{}, 0)

	err = json.Unmarshal(immutableDebtNoteJSON, &debtNoteInput)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	// Check the basic conditions on the parameter properties
	if len(getSafeString(debtNoteInput["debtNoteID"])) == 0 {
		return fmt.Errorf("debtNoteID field must be a non-empty string")
	}
	if len(getSafeString(debtNoteInput["debt"])) == 0 {
		return fmt.Errorf("debt field must be a greater than zero")
	}

	// Get client org id and verify it matches peer org id.
	// In this scenario, client is only authorized to read/write private data from its own peer.
	clientOrgID, err := getClientOrgID(ctx, true)
	if err != nil {
		return fmt.Errorf("failed to get verified OrgID: %v", err)
	}

	// Check if debtNote already exists in world state
	debtNoteEntry, err := ctx.GetStub().GetState(getSafeString(debtNoteInput["debtNoteID"]))
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	if debtNoteEntry != nil {
		return fmt.Errorf("%s already exists", getSafeString(debtNoteInput["debtNoteID"]))
	}

	// Check if debtNote already exists in private collection
	// Get the owner's implicit private data collection
	collection := buildCollectionName(clientOrgID)

	debtNoteAsBytes, err := ctx.GetStub().GetPrivateData(collection, getSafeString(debtNoteInput["debtNoteID"]))
	if err != nil {
		return fmt.Errorf("failed to get debtNote: %v", err)
	} else if debtNoteAsBytes != nil {
		fmt.Println("DebtNote already exists: " + getSafeString(debtNoteInput["debtNoteID"]))
		return fmt.Errorf("this DebtNote already exists in your private space: " + getSafeString(debtNoteInput["debtNoteID"]))
	}

	// update the debtNote properties with owner and debty details
	//debtNoteInput["ownerOrg"]=clientOrgID
	if getSafeString(debtNoteInput["debtyOrg"]) != clientOrgID {
		return fmt.Errorf("Not possible to create a debtNote in other organization name as debty")
	}

	//debtNoteInput["redeemStatus"]="False"

	// Prepare to store in owners private collection
	debtNoteBytes, err := json.Marshal(debtNoteInput)
	if err != nil {
		return fmt.Errorf("failed to create asset JSON: %v", err)
	}

	// Persist private immutable asset properties to owner's private data collection
	err = ctx.GetStub().PutPrivateData(collection, getSafeString(debtNoteInput["debtNoteID"]), debtNoteBytes)
	if err != nil {
		return fmt.Errorf("failed to put Asset private details: %v", err)
	}

	// record the debtnote into world state
	//calculate the hash of the private data
	hash := sha256.New()
	hash.Write(immutableDebtNoteJSON)
	calculatedHash := hash.Sum(nil)

	err = ctx.GetStub().PutState(getSafeString(debtNoteInput["debtNoteID"]), calculatedHash)
	if err != nil {
		return fmt.Errorf("failed to put asset in public data: %v", err)
	}

	return nil
}

// transferDebtNote performs the private state updates for the transferred debtNote
func (s *SmartContract) DeleteDebtNote(ctx contractapi.TransactionContextInterface) error {
        // Get client org id and verify it matches peer org id.
        // In this scenario, client is only authorized to read/write private data from its own peer.
        clientOrgID, err := getClientOrgID(ctx, false)
        if err != nil {
                return fmt.Errorf("failed to get verified OrgID: %v", err)
        }

        transientMap, err := ctx.GetStub().GetTransient()
        if err != nil {
                return fmt.Errorf("error getting transient: %v", err)
        }
	immutableDebtNoteJSON, ok := transientMap["debtnote_properties"]
        if !ok {
                return fmt.Errorf("debtyOrg key not found in the transient map")
        }

        // fetch the debt details
        //var debtNoteInput DebtNoteOwner

        var debtNoteInput map[string]interface{}
        debtNoteInput = make(map[string]interface{}, 0)

        err = json.Unmarshal(immutableDebtNoteJSON, &debtNoteInput)
        if err != nil {
                return fmt.Errorf("failed to unmarshal JSON: %v", err)
        }

        // Check the basic conditions on the new owner data
        if len(getSafeString(debtNoteInput["debtNoteID"])) == 0 {
                return fmt.Errorf("debtNoteID field must be a non-empty string")
        }

        // Check if debtNote already exists in private collection
        // Get the owner's implicit private data collection
        collection := buildCollectionName(clientOrgID)

        debtNoteAsBytes, err := ctx.GetStub().GetPrivateData(collection, getSafeString(debtNoteInput["debtNoteID"]))
        if err != nil {
                return fmt.Errorf("failed to get debtNote: %v", err)
        } else if debtNoteAsBytes == nil {
                fmt.Println("DebtNote does not exists: " + getSafeString(debtNoteInput["debtNoteID"]))
                return fmt.Errorf("this DebtNote does not exists in your private space: " + getSafeString(debtNoteInput["debtNoteID"]))
        }

        // Transfer the private properties (delete from current owner collection, create in new owner collection)
        // Get the owner's implicit private data collection
        //collectionCurrentOrg := buildCollectionName(clientOrgID)
        err = ctx.GetStub().DelPrivateData(collection, getSafeString(debtNoteInput["debtNoteID"]))
        if err != nil {
                return fmt.Errorf("failed to delete Asset private details from seller: %v", err)
        }
        return nil

}



// transferDebtNote performs the private state updates for the transferred debtNote
func (s *SmartContract) TransferDebtNote(ctx contractapi.TransactionContextInterface) error {

/*	// Get client org id and verify it matches peer org id.
	// In this scenario, client is only authorized to read/write private data from its own peer.
	clientOrgID, err := getClientOrgID(ctx, false)
	if err != nil {
		return fmt.Errorf("failed to get verified OrgID: %v", err)
	}
*/
	transientMap, err := ctx.GetStub().GetTransient()
	if err != nil {
		return fmt.Errorf("error getting transient: %v", err)
	}

	// Asset properties must be retrieved from the transient field as they are private
	immutableDebtNoteJSON, ok := transientMap["debtnote_new_owner"]
	if !ok {
		return fmt.Errorf("debtnote_new_owner key not found in the transient map")
	}

	immutableTransferableDebtNoteJSON, ok := transientMap["debtnote_newproperties"]
	if !ok {
		return fmt.Errorf("debtyOrg key not found in the transient map")
	}

	// fetch the debt details
	//var debtNoteInput DebtNoteOwner

	var debtNoteInput map[string]interface{}
	debtNoteInput = make(map[string]interface{}, 0)

	err = json.Unmarshal(immutableDebtNoteJSON, &debtNoteInput)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	// Check the basic conditions on the new owner data
	if len(getSafeString(debtNoteInput["debtNoteID"])) == 0 {
		return fmt.Errorf("debtNoteID field must be a non-empty string")
	}
	if len(getSafeString(debtNoteInput["newOwnerOrg"])) == 0 {
		return fmt.Errorf("debt field must be a greater than zero")
	}

	// Get the owner's implicit private data collection
	collectionNewOrg := buildCollectionName(getSafeString(debtNoteInput["newOwnerOrg"]))

	// Persist private immutable asset properties to owner's private data collection
	err = ctx.GetStub().PutPrivateData(collectionNewOrg, getSafeString(debtNoteInput["debtNoteID"]), immutableTransferableDebtNoteJSON)
	if err != nil {
		return fmt.Errorf("failed to put debtNote in new owners private details: %v", err)
	}
/*
	// Transfer the private properties (delete from current owner collection, create in new owner collection)
	// Get the owner's implicit private data collection
	collectionCurrentOrg := buildCollectionName(clientOrgID)
	err = ctx.GetStub().DelPrivateData(collectionCurrentOrg, getSafeString(debtNoteInput["debtNoteID"]))
	if err != nil {
		return fmt.Errorf("failed to delete Asset private details from seller: %v", err)
	}
*/
	// record the debtnote into world state
	//calculate the hash of the private data
	hash := sha256.New()
	hash.Write(immutableDebtNoteJSON)
	calculatedHash := hash.Sum(nil)

	err = ctx.GetStub().PutState(getSafeString(debtNoteInput["debtNoteID"])+"_TRANSFER", calculatedHash)
	if err != nil {
		return fmt.Errorf("failed to put asset in public data: %v", err)
	}

	return nil
}

// GetDebtNote returns the immutable asset properties from owner's private data collection
func (s *SmartContract) GetDebtNote(ctx contractapi.TransactionContextInterface, debtNoteID string) (string, error) {
	// In this scenario, client is only authorized to read/write private data from its own peer.
	collection, err := getClientImplicitCollectionName(ctx)
	if err != nil {
		return "", err
	}

	immutableProperties, err := ctx.GetStub().GetPrivateData(collection, debtNoteID)
	if err != nil {
		return "", fmt.Errorf("failed to read debtNote private properties from client org's collection: %v", err)
	}
	if immutableProperties == nil {
		return "", fmt.Errorf("debtNote private details does not exist in client org's collection: %s", debtNoteID)
	}

	return string(immutableProperties), nil
}

// GetDebtNoteHash returns the immutable Hash of the debt note from owner's private data collection
func (s *SmartContract) GetDebtNoteHash(ctx contractapi.TransactionContextInterface, debtNoteID string, orgID string) (string, error) {
	// In this scenario, client is only authorized to read/write private data from its own peer.

	collection := buildCollectionName(orgID)

	immutablePropertiesOnChainHash, err := ctx.GetStub().GetPrivateDataHash(collection, debtNoteID)
	if err != nil {
		return debtNoteID, fmt.Errorf("failed to read debtnote private properties hash from %s collection: %v", orgID, err)
	}
	if immutablePropertiesOnChainHash == nil {
		return debtNoteID, fmt.Errorf("asset private properties hash does not exist: %s for %s", debtNoteID, orgID)
	}

	returnStr := fmt.Sprintf("%x", immutablePropertiesOnChainHash)
	return returnStr, nil
}

// ReadWorldState returns the public data
func (s *SmartContract) ReadWorldState(ctx contractapi.TransactionContextInterface, key string) (string, error) {
	// Since only public data is accessed in this function, no access control is required
	valueJSON, err := ctx.GetStub().GetState(key)
	if err != nil {
		return key, fmt.Errorf("failed to read from world state: %v", err)
	}
	if valueJSON == nil {
		return key, fmt.Errorf("%s does not exist", key)
	}

	returnStr := fmt.Sprintf("%x", valueJSON)
	return returnStr, nil
}

// transferDebtNote performs the private state updates for the transferred debtNote
func (s *SmartContract) RedeemDebtNote(ctx contractapi.TransactionContextInterface) error {

	transientMap, err := ctx.GetStub().GetTransient()
	if err != nil {
		return fmt.Errorf("error getting transient: %v", err)
	}

	// Asset properties must be retrieved from the transient field as they are private
	immutableDebtNoteJSON, ok := transientMap["debtnote_redeem"]
	if !ok {
		return fmt.Errorf("debtyOrg key not found in the transient map")
	}

	// fetch the debt details
	//var debtNoteInput DebtNoteOwner
	var debtNoteInput map[string]interface{}
	debtNoteInput = make(map[string]interface{}, 0)

	err = json.Unmarshal(immutableDebtNoteJSON, &debtNoteInput)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	// Check the basic conditions on the new owner data
	if len(getSafeString(debtNoteInput["debtNoteID"])) == 0 {
		return fmt.Errorf("debtNoteID field must be a non-empty string")
	}

	// Get client org id and verify it matches peer org id.
	// In this scenario, client is only authorized to read/write private data from its own peer.
	clientOrgID, err := getClientOrgID(ctx, true)
	if err != nil {
		return fmt.Errorf("failed to get verified OrgID: %v", err)
	}

	// Check if debtNote already exists
	collection := buildCollectionName(clientOrgID)
	debtNoteAsBytes, err := ctx.GetStub().GetPrivateData(collection, getSafeString(debtNoteInput["debtNoteID"]))
	if err != nil {
		return fmt.Errorf("failed to get debtNote: %v", err)
	} else if debtNoteAsBytes == nil {
		fmt.Println("DebtNote does not exist: " + getSafeString(debtNoteInput["debtNoteID"]))
		return fmt.Errorf("this DebtNote does not exist: " + getSafeString(debtNoteInput["debtNoteID"]))
	}

	// Auth check to ensure that client's org actually owns the asset
	//var debtNoteAsBytesInput DebtNote

	var debtNoteAsBytesInput map[string]interface{}
	debtNoteAsBytesInput = make(map[string]interface{}, 0)

	err = json.Unmarshal(debtNoteAsBytes, &debtNoteAsBytesInput)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	if clientOrgID != getSafeString(debtNoteAsBytesInput["debtyOrg"]) {
		return fmt.Errorf("a client from %s cannot update the description of a asset owned by %s", clientOrgID, getSafeString(debtNoteAsBytesInput["ownerOrg"]))
	}
	if getSafeString(debtNoteAsBytesInput["redeemStatus"]) != "False" {
		return fmt.Errorf("the debnote %s already been redeemed by the owner %s ", getSafeString(debtNoteAsBytesInput["redeemStatus"]), getSafeString(debtNoteAsBytesInput["ownerOrg"]))
	}

	// update the debtNote properties with owner and debty details
	debtNoteAsBytesInput["redeemStatus"] = "True"

	// Prepare to store in owners private collection
	debtNoteBytes, err := json.Marshal(debtNoteAsBytesInput)
	if err != nil {
		return fmt.Errorf("failed to create asset JSON: %v", err)
	}

	// Persist private immutable asset properties to owner's private data collection
	err = ctx.GetStub().PutPrivateData(collection, getSafeString(debtNoteAsBytesInput["debtNoteID"]), debtNoteBytes)
	if err != nil {
		return fmt.Errorf("failed to put debtNote in new owners private details: %v", err)
	}

	return nil
}

// getClientOrgID gets the client org ID.
// The client org ID can optionally be verified against the peer org ID, to ensure that a client
// from another org doesn't attempt to read or write private data from this peer.
// The only exception in this scenario is for TransferAsset, since the current owner
// needs to get an endorsement from the buyer's peer.
func getClientOrgID(ctx contractapi.TransactionContextInterface, verifyOrg bool) (string, error) {
	clientOrgID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return "", fmt.Errorf("failed getting client's orgID: %v", err)
	}

	if verifyOrg {
		err = verifyClientOrgMatchesPeerOrg(clientOrgID)
		if err != nil {
			return "", err
		}
	}

	return clientOrgID, nil
}

// verifyClientOrgMatchesPeerOrg checks the client org id matches the peer org id.
func verifyClientOrgMatchesPeerOrg(clientOrgID string) error {
	peerOrgID, err := shim.GetMSPID()
	if err != nil {
		return fmt.Errorf("failed getting peer's orgID: %v", err)
	}

	if clientOrgID != peerOrgID {
		return fmt.Errorf("client from org %s is not authorized to read or write private data from an org %s peer",
			clientOrgID,
			peerOrgID,
		)
	}

	return nil
}

func buildCollectionName(clientOrgID string) string {
	return fmt.Sprintf("_implicit_org_%s", clientOrgID)
}

func getClientImplicitCollectionName(ctx contractapi.TransactionContextInterface) (string, error) {
	clientOrgID, err := getClientOrgID(ctx, true)
	if err != nil {
		return "", fmt.Errorf("failed to get verified OrgID: %v", err)
	}

	err = verifyClientOrgMatchesPeerOrg(clientOrgID)
	if err != nil {
		return "", err
	}

	return buildCollectionName(clientOrgID), nil
}

//check whether string has value or not
func getSafeString(input interface{}) string {
	var safeValue string
	var isOk bool

	if input == nil {
		safeValue = ""
	} else {
		safeValue, isOk = input.(string)
		if isOk == false {
			safeValue = ""
		}
	}
	return safeValue
}

func main() {
	chaincode, err := contractapi.NewChaincode(new(SmartContract))
	if err != nil {
		log.Panicf("Error create transfer asset chaincode: %v", err)
	}

	if err := chaincode.Start(); err != nil {
		log.Panicf("Error starting asset chaincode: %v", err)
	}
}

