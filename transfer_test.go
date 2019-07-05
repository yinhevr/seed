package seed

import "testing"

// TestTransfer ...
func TestTransfer(t *testing.T) {
	seed := NewSeed(DatabaseOption("sqlite3", "d:\\cs.db"), Transfer("D:\\workspace\\goproject\\seed\\old\\seed.db", InfoFlagSQLite, TransferStatusOld))
	seed.Workspace = "D:\\videoall"
	seed.Start()
	seed.Wait()
}
