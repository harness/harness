package migrate

import (
	"database/sql"
	"encoding/pem"
	"log"
)

type rev20140409095256 struct{}

type privekeynslug struct {
	PrivateKey string
	Slug       string
}

var ChangePrivkeyUseText = &rev20140409095256{}

func (r *rev20140409095256) Revision() int64 {
	return 20140409095256
}

func (r *rev20140409095256) Up(mg *MigrationDriver) error {
	if _, err := mg.ChangeColumn("repos", "private_key", "TEXT"); err != nil {
		return err
	}

	pks, err := fetchPrivateKeys(mg)
	if err != nil {
		return err
	}

	var ec int
	for _, pk := range pks {
		// Check if we can decode existing private key
		block, _ := pem.Decode([]byte(pk.PrivateKey))
		if block == nil {
			ec += 1
			log.Printf("Private key broken for: %s\n", pk.Slug)
		}
	}
	log.Printf("%d errors, %d keys total\n", ec, len(pks))

	return err
}

func (r *rev20140409095256) Down(mg *MigrationDriver) error {
	_, err := mg.ChangeColumn("repos", "private_key", "VARCHAR(1024)")
	return err
}

func fetchPrivateKeys(mg *MigrationDriver) ([]privekeynslug, error) {
	var result []privekeynslug
	rows, err := mg.Tx.Query(`select slug, private_key from repos`)
	if err != nil {

		return result, err
	}
	for rows.Next() {
		var slug, pk sql.NullString
		if err := rows.Scan(&slug, &pk); err != nil {
			return result, err
		}
		pks := privekeynslug{}
		if slug.Valid {
			pks.Slug = slug.String
		}
		if pk.Valid {
			pks.PrivateKey = pk.String
		}
		result = append(result, pks)
	}
	return result, err
}
