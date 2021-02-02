package db

import (
	"encoding/json"
	"fmt"
	"time"
)

//License struct representing the license object
type License struct {
	AccountID string `json:"accountId" binding:"required"`
	AppID     string `json:"appId" binding:"required"`
	UserID    string `json:"userId" binding:"required"`
	Features  string `json:"features" binding:"required"`
	ExpiresAt string `json:"expires,omitempty"`
}

//AccountReserved lists all reserved licenses for an account and appID
type AccountReserved struct {
	Count   int `json:"count"`
	License []*License
}

const licensePrefix string = "reserved:"

//CreateReservation reserves a license for app / user / account
func (db *Database) CreateReservation(license *License) error {
	ttl := time.Now().Add(30 * time.Second)
	license.ExpiresAt = ttl.Format(time.RFC3339)
	licenseJSON, err := json.Marshal(license)
	if err != nil {
		return err
	}
	fmt.Printf("JSON: %v", licenseJSON)
	pipe := db.Client.TxPipeline()
	pipe.HSet(licensePrefix+license.AccountID+license.AppID, license.UserID, string(licenseJSON)) // trying to use hashSet for storing the items..
	_, err = pipe.Exec()
	if err != nil {
		return err
	}

	return nil
}

//FetchAccountReservations fetches all of an account + app reservervations
func (db *Database) FetchAccountReservations(account string, appID string) (*AccountReserved, error) {
	reservationEntries := db.Client.HGetAll(licensePrefix + account + appID)
	if reservationEntries == nil {
		return nil, ErrNil
	}
	count := len(reservationEntries.Val())
	//create the new collection to be returned
	licenses := make([]*License, count)
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
		}
		//cleanup the expired records...
	}

	reservationEntryResponse := &AccountReserved{Count: idx, License: licenses}
	return reservationEntryResponse, nil

}

//FetchUserReservation fetch a user's reserved license if found and not expired.
func (db *Database) FetchUserReservation(userID string, appID string, accountID string) (*License, error) {
	userLicense := db.Client.HGet(licensePrefix+accountID+appID, userID)
	if userLicense == nil {
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
		//TODO: try to fetch a new free license?
		return nil, ErrNil
	}
	//TODO: need to increment the expiriation time of the reservation
	return &license, nil

}

//Checks if a license is expired based on ExpiresOn field, returns true if expired, along with the time.Time object
func (license *License) isLicenseExpired() (bool, time.Time, error) {
	ttl, err := time.Parse(time.RFC3339, license.ExpiresAt) //convert string to time
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

// //ReserveLicense saves a license to redis
// func (db *Database) ReserveLicense(license *License) error {
// 	//set the record expiration time from now
// 	ttl := time.Now().Add(10 * time.Minute)
// 	license.ExpiresAt = ttl.Local().Format("3:04PM")
// 	//serialize License obj to JSON
// 	licenseJSON, err := json.Marshal(license)
// 	if err != nil {
// 		return err
// 	}
// 	// Set Obj
// 	err = db.Client.Set(licensePrefix+license.UserID+license.AppID, licenseJSON, 0).Err()
// 	if err != nil {
// 		return err
// 	}
// 	//setting ttl on the inital set only takes duration, instead we set an exact time for expiration
// 	err = db.Client.ExpireAt(licensePrefix+license.UserID+license.AppID, ttl).Err()
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// //FetchReservedLicense returns records for a given userID if they exist
// func (db *Database) FetchReservedLicense(recordID string) (*License, error) {
// 	s, err := db.Client.Get(licensePrefix + recordID).Result()
// 	if err == redis.Nil {
// 		fmt.Printf("License for %v does not exist", recordID)
// 		return nil, ErrNil
// 	} else if err != nil {
// 		return nil, err
// 	}
// 	license := License{}
// 	err = json.Unmarshal([]byte(s), &license)
// 	//calculate new ttl from current time
// 	ttl := time.Now().Add(10 * time.Minute)
// 	//reup the expiration time by the duration of life
// 	err = db.Client.ExpireAt(licensePrefix+recordID, ttl).Err()
// 	if err != nil {
// 		return nil, err
// 	}
// 	//update the expiresAt with new ttl
// 	license.ExpiresAt = ttl.Local().Format("3:04PM")
// 	return &license, nil
// }
