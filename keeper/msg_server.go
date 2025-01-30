package keeper

import (
	"context"
	"log"
	"slices"
	"strconv"
	"strings"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ipfs/go-cid"

	"github.com/janction/videoRendering"
	"github.com/janction/videoRendering/ipfs"
)

type msgServer struct {
	k Keeper
}

var _ videoRendering.MsgServer = msgServer{}

// NewMsgServerImpl returns an implementation of the module MsgServer interface.
func NewMsgServerImpl(keeper Keeper) videoRendering.MsgServer {
	return &msgServer{k: keeper}
}

// CreateGame defines the handler for the MsgCreateVideoRenderingTask message.
func (ms msgServer) CreateVideoRenderingTask(ctx context.Context, msg *videoRendering.MsgCreateVideoRenderingTask) (*videoRendering.MsgCreateVideoRenderingTaskResponse, error) {
	// TODO had validations about the parameters
	taskInfo, err := ms.k.VideoRenderingTaskInfo.Get(ctx)
	if err != nil {
		return nil, err
	}

	// we make sure the provided CID is valid
	_, err = cid.Decode(msg.Cid)
	if err != nil {
		log.Printf("provided cid is invalid: (%s)", msg.Cid)
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidVideoRenderingTask.Error(), "cid %s is invalid", msg.Cid)
	}

	var nextId = taskInfo.NextId
	// we get the taskId in string
	taskId := strconv.FormatUint(nextId, 10)

	// and increase the task id counter for next task
	nextId++
	ms.k.VideoRenderingTaskInfo.Set(ctx, videoRendering.VideoRenderingTaskInfo{NextId: nextId})

	videoTask := videoRendering.VideoRenderingTask{TaskId: taskId, Requester: msg.Creator, Cid: msg.Cid, StartFrame: msg.StartFrame, EndFrame: msg.EndFrame, Completed: false, ThreadAmount: msg.Threads, Reward: msg.Reward}
	threads := videoTask.GenerateThreads(taskId)
	videoTask.Threads = threads

	if err := ms.k.VideoRenderingTasks.Set(ctx, taskId, videoTask); err != nil {
		return nil, err
	}
	return &videoRendering.MsgCreateVideoRenderingTaskResponse{TaskId: taskId}, nil
}

func (ms msgServer) AddWorker(ctx context.Context, msg *videoRendering.MsgAddWorker) (*videoRendering.MsgAddWorkerResponse, error) {
	found, err := ms.k.Workers.Has(ctx, msg.Creator)
	if err != nil {
		return nil, err
	}

	if found {
		log.Printf("Worker %v already exists.", msg.Creator)
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrWorkerAlreadyRegistered.Error(), "worker (%s) is already registered", msg.Creator)
	}

	// worker is not previously registered, so we move on
	// TODO I'm facking a stacked value of 100 for future use
	reputation := videoRendering.Worker_Reputation{Points: 0, Stacked: 100}
	worker := videoRendering.Worker{Address: msg.Creator, Reputation: &reputation, Enabled: true}

	ms.k.Workers.Set(ctx, msg.Creator, worker)
	return &videoRendering.MsgAddWorkerResponse{}, nil
}

func (ms msgServer) SubscribeWorkerToTask(ctx context.Context, msg *videoRendering.MsgSubscribeWorkerToTask) (*videoRendering.MsgSubscribeWorkerToTaskResponse, error) {
	worker, err := ms.k.Workers.Get(ctx, msg.Address)
	if err != nil {
		return nil, err
	}

	if !worker.Enabled {
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrWorkerNotAvailable.Error(), "worker (%s) it nos enabled or doesn't exists", msg.Address)
	}
	task, err := ms.k.VideoRenderingTasks.Get(ctx, msg.TaskId)
	if err != nil {
		return nil, err
	}
	if task.Completed {
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrWorkerTaskNotAvailable.Error(), "task (%s) is already completed. Can't subscribe worker", msg.TaskId)
	}

	MAX_WORKERS_PER_THREAD := 2
	for i, v := range task.Threads {
		// TODO MaxWorkersPerThread value should be global
		if len(v.Workers) < MAX_WORKERS_PER_THREAD && !v.Completed {
			v.Workers = append(v.Workers, msg.Address)

			ms.k.VideoRenderingTasks.Set(ctx, task.TaskId, task)
			worker.CurrentTaskId = task.TaskId
			worker.CurrentThreadIndex = uint32(i)
			ms.k.Workers.Set(ctx, msg.Address, worker)

			err := ms.k.VideoRenderingTasks.Set(ctx, task.TaskId, task)
			if err != nil {
				log.Printf("error trying to update thread %s to in progress", v.ThreadId)
			}
			return &videoRendering.MsgSubscribeWorkerToTaskResponse{ThreadId: v.ThreadId}, nil
		}
	}
	return nil, nil
}

func (ms msgServer) ProposeSolution(ctx context.Context, msg *videoRendering.MsgProposeSolution) (*videoRendering.MsgProposeSolutionResponse, error) {
	// creator of the solution must be a valid worker
	worker, err := ms.k.Workers.Get(ctx, msg.Creator)
	if err != nil {
		return &videoRendering.MsgProposeSolutionResponse{}, err
	}

	if !worker.Enabled {
		log.Printf("workers %s is not enabled to propose a solution", msg.Creator)
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidSolution.Error(), "workers %s is not enabled to propose a solution", msg.Creator)
	}

	task, err := ms.k.VideoRenderingTasks.Get(ctx, msg.TaskId)
	if err != nil {
		return nil, err
	}

	// task must exists and be in progress
	if task.Completed {
		log.Printf("Task %s is not valid to accept solutions", msg.TaskId)
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidSolution.Error(), "Task %s is not valid to accept solutions", msg.TaskId)
	}

	for i, v := range task.Threads {
		// TODO threads might be better as map instead of slice
		if v.ThreadId == msg.ThreadId {
			if v.Solution != nil {
				log.Printf("thread %s already has a solution", msg.ThreadId)
				return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidSolution.Error(), "thread %s already has a solution", msg.ThreadId)
			}
			// worker must be a valid registered worker in the thread with a solution
			if !slices.Contains(v.Workers, msg.Creator) {
				log.Printf("Worker %s is not valid at thread %s", msg.Creator, msg.ThreadId)
				return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidSolution.Error(), "Worker %s is not valid at thread %s", msg.Creator, msg.ThreadId)
			}

			// solution len must be equal to the frames generated
			if len(msg.Solution) != (int(v.EndFrame) - int(v.StartFrame) + 1) {
				log.Printf("amount of files in solution is incorrect, %v ", len(msg.Solution))
				return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidSolution.Error(), "amount of files in solution is incorrect, %v ", len(msg.Solution))
			}

			// we have passed all validations, lets add the solution to the thread
			parsedSolution := make(map[string]string)
			for _, pair := range msg.Solution {
				parts := strings.SplitN(pair, "=", 2)
				if len(parts) != 2 {
					log.Printf("invalid solution format; expected key=value")
					return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidSolution.Error(), "invalid solution format; expected key=value")
				}
				parsedSolution[parts[0]] = parts[1]
			}

			// TODO parse file names equals thread's frames

			task.Threads[i].Solution = &videoRendering.VideoRenderingThread_Solution{ProposedBy: msg.Creator, Hashes: msg.Solution}
			err = ms.k.VideoRenderingTasks.Set(ctx, msg.TaskId, task)
			if err != nil {
				log.Printf("unable to propose solution %s", err.Error())
				return nil, err
			}
			log.Printf("Proposing solution %s", msg.Solution)
		}
	}

	return &videoRendering.MsgProposeSolutionResponse{}, nil
}

func (ms msgServer) SubmitValidation(ctx context.Context, msg *videoRendering.MsgSubmitValidation) (*videoRendering.MsgSubmitValidationResponse, error) {
	// validation must be from a worker on the thread
	task, err := ms.k.VideoRenderingTasks.Get(ctx, msg.TaskId)

	if err != nil {
		return nil, err
	}

	worker, err := ms.k.Workers.Get(ctx, msg.Creator)

	if err != nil {
		return nil, err
	}

	if !worker.Enabled {
		log.Printf("worker is not allowed to validate solutions")
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidVerification.Error(), "worker is not allowed to validate solutions")
	}

	if worker.CurrentTaskId != msg.TaskId {
		log.Printf("worker is not working on task")
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidVerification.Error(), "worker is not working on task")
	}

	if task.Completed {
		log.Printf("task is already completed. No more validations accepted")
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidVerification.Error(), "task is already completed. No more validations accepted")
	}

	thread := task.Threads[worker.CurrentThreadIndex]
	if thread.ThreadId != msg.ThreadId {
		log.Printf("worker is not working on thread")
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidVerification.Error(), "worker is not working on thread")
	}

	// this shouldn't happen.
	if !slices.Contains(thread.Workers, msg.Creator) {
		log.Printf("worker is not working on thread")
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidVerification.Error(), "worker is not working on thread")
	}

	validation := videoRendering.VideoRenderingThread_Validation{Validator: msg.Creator, AmountFiles: msg.FilesAmount, Valid: msg.Valid}
	task.Threads[worker.CurrentThreadIndex].Validations = append(thread.Validations, &validation)
	ms.k.VideoRenderingTasks.Set(ctx, msg.TaskId, task)

	return &videoRendering.MsgSubmitValidationResponse{}, nil
}

func (ms msgServer) SubmitSolution(ctx context.Context, msg *videoRendering.MsgSubmitSolution) (*videoRendering.MsgSubmitSolutionResponse, error) {
	task, err := ms.k.VideoRenderingTasks.Get(ctx, msg.TaskId)
	if err != nil {
		return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidSolution.Error(), "provided task doesn't exists")
	}
	for i, thread := range task.Threads {
		if thread.ThreadId == msg.ThreadId {

			if thread.Solution.ProposedBy != msg.Creator {
				return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidSolution.Error(), "only the provider of the solution can upload it")
			}

			if !thread.Completed {
				return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidSolution.Error(), "thread is not yet completed")
			}

			// we make sure ipfs is running
			ipfs.EnsureIPFSRunning()

			// we verify the solution
			err := thread.VerifySubmittedSolution(msg.Cid)
			if err != nil {
				return nil, sdkerrors.ErrAppConfig.Wrapf(videoRendering.ErrInvalidSolution.Error(), "submited solution is incorrect")
			}

			task.Threads[i].Solution.Files = msg.Cid
			ms.k.VideoRenderingTasks.Set(ctx, msg.TaskId, task)
			break
		}
	}
	return &videoRendering.MsgSubmitSolutionResponse{}, nil
}
