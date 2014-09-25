package blobsql

type Blob struct {
	ID   int64  `meddler:"blob_id,pk"        orm:"column(blob_id);pk;auto"`
	Path string `meddler:"blob_path"         orm:"column(blob_path);size(2000);unique"`
	Data string `meddler:"blob_data,gobgzip" orm:"column(blob_data);type(text)"`
}

func (b *Blob) TableName() string { return "blobs" }
