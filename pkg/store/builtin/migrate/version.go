package migrate

import (
	"github.com/BurntSushi/migration"
)

// GetVersion gets the migration version from the database,
// creating the migration table if it does not already exist.
func GetVersion(tx migration.LimitedTx) (int, error) {
	v, err := getVersion(tx)
	if err != nil {
		if err := createVersionTable(tx); err != nil {
			return 0, err
		}
		return getVersion(tx)
	}
	return v, nil
}

// SetVersion sets the migration version in the database,
// creating the migration table if it does not already exist.
func SetVersion(tx migration.LimitedTx, version int) error {
	if err := setVersion(tx, version); err != nil {
		if err := createVersionTable(tx); err != nil {
			return err
		}
		return setVersion(tx, version)
	}
	return nil
}

// setVersion updates the migration version in the database.
func setVersion(tx migration.LimitedTx, version int) error {
	_, err := tx.Exec(rebind("UPDATE migration_version SET version = ?"), version)
	return err
}

// getVersion gets the migration version in the database.
func getVersion(tx migration.LimitedTx) (int, error) {
	var version int
	row := tx.QueryRow("SELECT version FROM migration_version")
	if err := row.Scan(&version); err != nil {
		return 0, err
	}
	return version, nil
}

// createVersionTable creates the version table and inserts the
// initial value (0) into the database.
func createVersionTable(tx migration.LimitedTx) error {
	_, err := tx.Exec("CREATE TABLE migration_version ( version INTEGER )")
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT INTO migration_version (version) VALUES (0)")
	return err
}
