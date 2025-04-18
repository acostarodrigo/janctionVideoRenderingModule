package videoRendering

import (
	"context"
	fmt "fmt"
	"strconv"
	"testing"

	"bou.ke/monkey"
	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/janction/videoRendering/mocks"

	"github.com/stretchr/testify/require"
)

// --- Test for Validate ---
func TestValidate(t *testing.T) {
	task := &VideoRenderingTask{
		TaskId:       "task1",
		Requester:    "user1",
		Cid:          "QmTestCid",
		StartFrame:   0,
		EndFrame:     9,
		ThreadAmount: 2,
		Completed:    false,
		Reward:       &types.Coin{Denom: "token", Amount: sdkmath.NewInt(1000)},
	}

	err := task.Validate()
	require.NoError(t, err)
}

// --- Test for GetEmptyVideoRenderingTaskList ---
func TestGetEmptyVideoRenderingTaskList(t *testing.T) {
	list := GetEmptyVideoRenderingTaskList()
	require.Equal(t, []IndexedVideoRenderingTask{}, list)
}

// --- Test for GenerateThreads ---
func TestGenerateThreads(t *testing.T) {
	task := &VideoRenderingTask{
		TaskId:       "task1",
		Requester:    "user1",
		Cid:          "QmTestCid",
		StartFrame:   0,
		EndFrame:     9,
		ThreadAmount: 2,
		Completed:    false,
		Reward:       &types.Coin{Denom: "token", Amount: sdkmath.NewInt(1000)},
	}

	threads := task.GenerateThreads(task.TaskId)
	require.Len(t, threads, int(task.ThreadAmount))

	expectedRanges := []struct {
		start int64
		end   int64
	}{
		{0, 4},
		{5, 9},
	}

	for i, thread := range threads {
		expectedID := task.TaskId + strconv.Itoa(i)
		require.Equal(t, expectedID, thread.ThreadId)
		require.Equal(t, expectedRanges[i].start, thread.StartFrame)
		require.Equal(t, expectedRanges[i].end, thread.EndFrame)
		require.Equal(t, task.TaskId, thread.TaskId)
	}
}

// --- Test for splitFrames ---
func TestSplitFrames(t *testing.T) {
	t.Run("Even number of frames: 10 frames, 2 threads", func(t *testing.T) {
		expected := []frameRange{
			{StartFrame: 1, EndFrame: 5},
			{StartFrame: 6, EndFrame: 10},
		}
		result := splitFrames(1, 10, 2)
		require.Equal(t, expected, result)
	})

	t.Run("Odd number of frames: 10 frames, 3 threads", func(t *testing.T) {
		expected := []frameRange{
			{StartFrame: 1, EndFrame: 4},
			{StartFrame: 5, EndFrame: 7},
			{StartFrame: 8, EndFrame: 10},
		}
		result := splitFrames(1, 10, 3)
		require.Equal(t, expected, result)
	})
}

// --- Test for SubscribeWorkerToTask ---
func TestSubscribeWorkerToTaskKo(t *testing.T) {
	mockDB := new(mocks.DB)
	task := VideoRenderingTask{}

	mockDB.On("UpdateTask", "task123", "thread456", false).Return(nil)

	patch := monkey.Patch(ExecuteCli, func(args []string) error {
		return fmt.Errorf("mock error")
	})
	defer patch.Unpatch()

	err := task.SubscribeWorkerToTask(context.Background(), "workerAddress", "task123", "thread456", mockDB)
	require.Error(t, err)
	mockDB.AssertExpectations(t)
}

func TestSubscribeWorkerToTaskOk(t *testing.T) {
	mockDB := new(mocks.DB)
	task := VideoRenderingTask{}

	patch := monkey.Patch(ExecuteCli, func(args []string) error {
		return nil
	})
	defer patch.Unpatch()

	err := task.SubscribeWorkerToTask(context.Background(), "workerAddress", "task123", "thread456", mockDB)
	require.NoError(t, err)
}

// --- Test for GetWinnerReward ---
func TestGetWinnerReward(t *testing.T) {
	task := VideoRenderingTask{
		Reward: &types.Coin{Denom: "token", Amount: sdkmath.NewInt(1000)},
		Threads: []*VideoRenderingThread{
			{}, {}, {}, {},
		},
	}

	reward := task.GetWinnerReward()

	expected := types.NewCoin("token", sdkmath.NewInt(1000).QuoRaw(2).QuoRaw(4))
	require.Equal(t, expected, reward)
}

// --- Test for GetValidatorsReward ---
func TestGetValidatorsReward(t *testing.T) {
	task := VideoRenderingTask{
		Reward: &types.Coin{Denom: "token", Amount: sdkmath.NewInt(1000)},
		Threads: []*VideoRenderingThread{
			{}, {}, {}, {},
		},
	}

	reward := task.GetValidatorsReward()

	expected := types.NewCoin("token", sdkmath.NewInt(1000).QuoRaw(2).QuoRaw(4))
	require.Equal(t, expected, reward)
}
