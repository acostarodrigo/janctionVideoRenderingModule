package videoRendering

import (
	"context"
	"log"

	"github.com/janction/videoRendering/ipfs"
	"github.com/janction/videoRendering/vm"
)

func (t *VideoRenderingThread) StartWork(worker string, cid string, path string) error {
	ctx := context.Background()

	isRunning := vm.IsContainerRunning(ctx, t.ThreadId)
	if !isRunning {
		// task is not running,
		// it could have finished? before we start it we check if we already have all the files to propose a solution
		count, err := vm.CountFilesInDirectory(path)
		if err != nil {
			return err
		}
		if count == (int(t.EndFrame)-int(t.StartFrame))+1 {
			// we have a solution!!
			log.Printf("Solution submitted for thread %s", t.ThreadId)
			return nil

		} else {
			// we don't have a solution, start working
			err = ipfs.IPFSGet(cid, path)
			if err != nil {
				log.Printf("Error getting cid %s", cid)
				return err
			}

			err = vm.RenderVideoThread(ctx, cid, uint64(t.StartFrame), uint64(t.EndFrame), t.ThreadId, path)
			if err != nil {
				return err
			}

		}
	} else {
		// Container is running, so we update worker status
		// if worker status is idle, we change it
		log.Printf("Work for thread %s is already going", t.ThreadId)

	}

	return nil
}
