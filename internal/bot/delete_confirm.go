package bot

import (
	"fmt"
	"sync"
)

const deleteRequiredPresses = 10

var deleteStore = struct {
	sync.Mutex
	m map[string]int
}{m: make(map[string]int)}

func deleteKey(chatID int64, msgID int) string {
	return fmt.Sprintf("%d:%d", chatID, msgID)
}

// pressDelete returns remaining presses after this press.
// First press sets counter to deleteRequiredPresses-1, subsequent presses decrement.
func pressDelete(chatID int64, msgID int) int {
	key := deleteKey(chatID, msgID)
	deleteStore.Lock()
	defer deleteStore.Unlock()

	remaining, exists := deleteStore.m[key]
	if !exists {
		remaining = deleteRequiredPresses - 1
	} else {
		remaining--
	}
	if remaining <= 0 {
		delete(deleteStore.m, key)
		return 0
	}
	deleteStore.m[key] = remaining
	return remaining
}
