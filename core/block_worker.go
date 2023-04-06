package core

type Worker struct {
	jobQueue chan interface{}
	jobFunc  func(interface{})
}

type TxWorker struct {
	jobQueue chan interface{}
	jobFunc  func()
}

func ExecuteTxWorker(queueSize int, workFunc func()) *TxWorker {
	return &TxWorker{
		jobQueue: make(chan interface{}, queueSize),
		jobFunc:  workFunc,
	}
}

func UpdateAssetTreeWorker(queueSize int, workFunc func(interface{})) *Worker {
	return &Worker{
		jobQueue: make(chan interface{}, queueSize),
		jobFunc:  workFunc,
	}
}

func UpdateAccountAndNftTreeWorker(queueSize int, workFunc func(interface{})) *Worker {
	return &Worker{
		jobQueue: make(chan interface{}, queueSize),
		jobFunc:  workFunc,
	}
}

func PreSaveBlockDataWorker(queueSize int, workFunc func(interface{})) *Worker {
	return &Worker{
		jobQueue: make(chan interface{}, queueSize),
		jobFunc:  workFunc,
	}
}

func SaveBlockDataWorker(queueSize int, workFunc func(interface{})) *Worker {
	return &Worker{
		jobQueue: make(chan interface{}, queueSize),
		jobFunc:  workFunc,
	}
}

func FinalSaveBlockDataWorker(queueSize int, workFunc func(interface{})) *Worker {
	return &Worker{
		jobQueue: make(chan interface{}, queueSize),
		jobFunc:  workFunc,
	}
}

func UpdatePoolTxWorker(queueSize int, workFunc func(interface{})) *Worker {
	return &Worker{
		jobQueue: make(chan interface{}, queueSize),
		jobFunc:  workFunc,
	}
}

func SyncAccountToRedisWorker(queueSize int, workFunc func(interface{})) *Worker {
	return &Worker{
		jobQueue: make(chan interface{}, queueSize),
		jobFunc:  workFunc,
	}
}

func (w *Worker) Enqueue(workDto interface{}) {
	w.jobQueue <- workDto
}

func (w *TxWorker) Enqueue(workDto interface{}) {
	w.jobQueue <- workDto
}

func (w *Worker) GetJobQueue() chan interface{} {
	return w.jobQueue
}

func (w *TxWorker) GetJobQueue() chan interface{} {
	return w.jobQueue
}

func (w *Worker) GetQueueSize() int {
	return len(w.jobQueue)
}

func (w *TxWorker) GetQueueSize() int {
	return len(w.jobQueue)
}

func (w *Worker) Start() {
	go func() {
		for workDto := range w.jobQueue {
			w.jobFunc(workDto)
		}
	}()
}

func (w *TxWorker) Start() {
	go func() {
		w.jobFunc()
	}()
}

func (w *Worker) Stop() {
	close(w.jobQueue)
}

func (w *TxWorker) Stop() {
	close(w.jobQueue)
}
