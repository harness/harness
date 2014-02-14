package database

import (
	. "github.com/drone/drone/pkg/model"
	"github.com/russross/meddler"
)

// Name of the Settings table in the database
const settingsTable = "settings"

// SQL Queries to retrieve the system settings
const settingsStmt = `
SELECT id, github_key, github_secret, github_domain, github_apiurl, bitbucket_key, bitbucket_secret,
smtp_server, smtp_port, smtp_address, smtp_username, smtp_password, hostname, scheme
FROM settings WHERE id = 1
`

//var (
//	// mutex for locking the local settings cache
//	settingsLock sync.Mutex
//
//	// cached settings
//	settingsCache = &Settings{}
//)

// Returns the system Settings.
func GetSettings() (*Settings, error) {
	//settingsLock.Lock()
	//defer settingsLock.Unlock()

	// return a copy of the settings
	//if settingsCache.ID == 0 {
	///	settingsCopy := &Settings{}
	//	*settingsCopy = *settingsCache
	//	return settingsCopy, nil
	//}

	settings := Settings{}
	err := meddler.QueryRow(db, &settings, settingsStmt)
	//if err == sql.ErrNoRows {
	//	// we ignore the NoRows error in case this
	//	// is the first time the system is being used
	//	err = nil
	//}
	return &settings, err
}

// Returns the system Settings. This is expected
// always pass, and will panic on failure.
func SettingsMust() *Settings {
	settings, err := GetSettings()
	if err != nil {
		panic(err)
	}
	return settings
}

// Saves the system Settings.
func SaveSettings(settings *Settings) error {
	//settingsLock.Lock()
	//defer settingsLock.Unlock()

	// persist changes to settings
	err := meddler.Save(db, settingsTable, settings)
	if err != nil {
		return err
	}

	// store updated settings in cache
	//*settingsCache = *settings
	return nil
}
