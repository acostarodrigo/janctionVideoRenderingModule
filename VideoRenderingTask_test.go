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

func TestSubscribeWorkerToTask(t *testing.T) {
	mockDB := new(mocks.DB)
	mockDB.On("UpdateTask", "task123", "thread456", false).Return(nil)

	patch := monkey.Patch(ExecuteCli, func(args []string) error {
		return fmt.Errorf("mock error")
	})
	defer patch.Unpatch()

	task := VideoRenderingTask{}
	err := task.SubscribeWorkerToTask(context.Background(), "workerAddress", "task123", "thread456", mockDB)
	require.Error(t, err)
	mockDB.AssertExpectations(t)
}

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

func TestGetValidatorsReward(t *testing.T) {
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
