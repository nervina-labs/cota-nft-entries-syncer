package data

import (
	"encoding/hex"
	"hash/crc32"

	"github.com/nervina-labs/cota-nft-entries-syncer/internal/biz"
	"github.com/nervina-labs/cota-nft-entries-syncer/internal/data/blockchain"
)

func GenerateSenderLock(entry biz.Entry) (lockScript biz.Script, err error) {
	hashType, err := entry.LockScript.HashType.Serialize()
	if err != nil {
		return
	}
	lockScript = biz.Script{
		CodeHash: entry.LockScript.CodeHash.String()[2:],
		HashType: hex.EncodeToString(hashType),
		Args:     hex.EncodeToString(entry.LockScript.Args),
	}
	return
}

func GenerateLockHash(entry biz.Entry) (lockHashStr string, lockHashCRC32 uint32, err error) {
	lockHash, err := entry.LockScript.Hash()
	if err != nil {
		return
	}
	lockHashStr = lockHash.String()[2:]
	lockHashCRC32 = crc32.ChecksumIEEE([]byte(lockHashStr))
	return
}

func GenerateReceiverLock(slice []byte) biz.Script {
	receiverLock := blockchain.ScriptFromSliceUnchecked(slice)
	script := biz.Script{
		CodeHash: hex.EncodeToString(receiverLock.CodeHash().RawData()),
		HashType: hex.EncodeToString(receiverLock.HashType().AsSlice()),
		Args:     hex.EncodeToString(receiverLock.Args().RawData()),
	}
	return script
}
