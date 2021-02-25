package db

import (
	"encoding/json"
	"fmt"
	"time"
)

//RecordDuration lifetime of records..
const RecordDuration = 10 * time.Second

//AccountReserved lists all reserved licenses for an account and appID
type AccountReserved struct {
	Count   int `json:"count"`
	License []*License
}

//License struct representing the license object
type License struct {
	AccountID string `json:"accountId" binding:"required"`
	AppID     string `json:"appId" binding:"required"`
	UserID    string `json:"userId" binding:"required"`
	ExpiresAt string `json:"expires,omitempty"`
}

//getPrimaryRecordKey returns the primary key for the set
func (license *License) getPrimaryRecordKey() string {
	return license.AccountID + "#" + license.AppID
}

//getTTLTime converts string time into time object
func (license *License) getTTLTime() (time.Time, error) {
	duration, err := time.Parse(time.RFC3339, license.ExpiresAt)
	if err != nil {
		return time.Time{}, err
	}
	return duration, nil
}

//IsLicenseExpired checks if a license is expired based on ExpiresOn field, returns true if expired, along with the time.Time object
func (license *License) isLicenseExpired() (bool, time.Time, error) {
	ttl, err := license.getTTLTime() //convert string to time
	if err != nil {
		return true, time.Time{}, err
	}
	timeDiff := time.Now().Sub(ttl)

	if timeDiff < 0 { //if positive ttl is in the past
		return false, ttl, nil //license not expired
	}
	//time.Time{} returns ZeroTime, which is nil time
	return true, time.Time{}, nil //license is expired

}

//IncrementExpirationTimeOnSetRecord bumps the expiration time, and updates the set record
func (license *License) incrementExpirationTimeOnSetRecord(db *Database) (bool, error) {
	ttl, err := license.getTTLTime() //convert string to time
	if err != nil {
		return false, err
	}
	newTTL := ttl.Add(RecordDuration) //increment time
	license.ExpiresAt = newTTL.Format(time.RFC3339)
	jsonLicense, err := license.marshalToJSON()
	if err != nil {
		return false, err
	}

	pipe := db.Client.TxPipeline()
	pipe.HSet(license.getPrimaryRecordKey(), license.UserID, jsonLicense)
	_, err = pipe.Exec()
	if err != nil {
		return false, err
	}

	return true, nil

}

//MarshalToJSON converts the license to json, and returns the byte array
func (license *License) marshalToJSON() ([]byte, error) {
	jsonObject, err := json.Marshal(license)
	if err != nil {
		return nil, err
	}
	return jsonObject, nil
}

//CreateReservation reserves a license for app / user / account returns true if successful
func (db *Database) CreateReservation(license *License) (*License, error) {
	ttl := time.Now().Add(RecordDuration)
	license.ExpiresAt = ttl.Format(time.RFC3339)
	licenseJSON, err := json.Marshal(license)
	if err != nil {
		return nil, err
	}
	pipe := db.Client.TxPipeline()
	pipe.HSet(license.getPrimaryRecordKey(), license.UserID, string(licenseJSON)) // trying to use hashSet for storing the items..
	fmt.Printf("user: %v pK: %v \n\n",license.UserID,license.getPrimaryRecordKey())
	_, err = pipe.Exec()
	if err != nil {
		return nil, err
	}

	return license, nil
}

//FetchAccountReservations fetches all of an account + app reservervations
func (db *Database) FetchAccountReservations(account string, appID string) (*AccountReserved, error) {
	reservationEntries := db.Client.HGetAll(account + "#" + appID)
	if reservationEntries == nil {
		return nil, ErrNil
	}
	count := len(reservationEntries.Val())
	//create the new collection to be returned
	licenses := make([]*License, count)
	//expired licenses
	var expiredLicense []*License
	//index of the collection
	idx := 0
	for _, value := range reservationEntries.Val() {
		//placeholder object for unmarshal
		licenseObj := License{}
		//unmarshal the value of the object in the hashset
		err := json.Unmarshal([]byte(value), &licenseObj)
		if err != nil {
			return nil, err
		}
		//convert the json.number type to int64
		isExpired, _, err := licenseObj.isLicenseExpired()
		if err != nil {
			return nil, err
		}
		if !isExpired {
			//push it into the collection
			licenses[idx] = &licenseObj
			idx++
		} else {
			expiredLicense = append(expiredLicense, &licenseObj)
		}
	}
	//If we have some expired licenses prune them
	if(len(expiredLicense) > 0){
		//cleanup the expired records...
		db.expireReservedLicenses(expiredLicense)
		expiredLicense = nil
	}
	if idx == 0 {
		return nil, ErrNil
	}
	reservationEntryResponse := &AccountReserved{Count: idx, License: licenses}
	return reservationEntryResponse, nil

}

//FetchUserReservation fetch a user's reserved license if found and not expired.
func (db *Database) FetchUserReservation(userID string, appID string, accountID string) (*License, error) {
	userLicense := db.Client.HGet(accountID+"#"+appID, userID)
	if userLicense.Val() == "" {
		return nil, ErrNil
	}
	license := License{}
	err := json.Unmarshal([]byte(userLicense.Val()), &license)
	if err != nil {
		return nil, err
	}
	fmt.Printf("License Found: %v\n", userLicense)
	isExpired, _, err := license.isLicenseExpired()
	if err != nil {
		return nil, err
	}
	if isExpired { //license is expired, just return not found
		fmt.Printf("License was expired attempting refresh.... on %v \n\n", license)
		db.expireReservedLicense(&license) // remove the expired license from the db
		licenseObj, err := db.CreateReservation(&license) //try to get a new license from a copy of the old one
		if(licenseObj == nil || err != nil){ //if failed or there was an error bail out
			fmt.Printf("After Refresh Attempt %v", err)
			return nil, err
		}
		fmt.Printf("License was successfully renewed: %v", licenseObj)
		return licenseObj, nil //return the new license
	}
	// increment the expiriation time of the reservation
	_, err = license.incrementExpirationTimeOnSetRecord(db) //increment the expiration time of the found active license
	if err != nil {
		return nil, err
	}

	return &license, nil

}
//ReturnUserLicense allows the user to unclaim a claimed license.
func (db *Database) ReturnUserLicense(userID string, appID string, accountID string) error {
	userLicense := db.Client.HGet(accountID+"#"+appID,userID) //pull the user's license from the hset
	if userLicense.Val() == "" {
		return ErrNil
	}
	license:= License{}
	err:= json.Unmarshal([]byte(userLicense.Val()), &license) //convert to License interface for easier use
	if err != nil {
		return err
	}
	_,err = db.Client.HDel(license.getPrimaryRecordKey(),license.UserID).Result()
	if err != nil {
		return err
	}

	return nil
}

//expiredReservedLicenses removes licenses from the redis set
func (db *Database) expireReservedLicenses(expired []*License) error {
	pipe := db.Client.TxPipeline()
	for _, value := range expired {
		fmt.Printf("Removing: %v\n\n", value)
		pipe.HDel(value.getPrimaryRecordKey(), value.UserID)
	}
	_, err := pipe.Exec()
	if err != nil {
		return err
	}

	return nil
}

func (db *Database) expireReservedLicense(license *License) error {
	pipe := db.Client.TxPipeline()
	pipe.HDel(license.getPrimaryRecordKey(),license.UserID)

	_, err := pipe.Exec()
	if err != nil {
		return err
	}

	return nil

}
