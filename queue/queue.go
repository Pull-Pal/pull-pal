package queue

import (
	"sync"

	"github.com/mobyvb/pull-pal/vc"
	"go.uber.org/zap"
)

type TaskType int

var (
	CommentTask TaskType = 0
	IssueTask   TaskType = 1
)

type Task struct {
	TaskType TaskType
	Issue    vc.Issue
	Comment  vc.Comment
}

type TaskQueue struct {
	log *zap.Logger
	// lockedIssues defines issues that are already accounted for in the queue
	lockedIssues map[int]bool
	// lockedPRs defines pull requests that are already accounted for in the queue
	lockedPRs map[int]bool
	queue     chan Task
	mu        sync.Mutex
}

func NewTaskQueue(log *zap.Logger, queueSize int) *TaskQueue {
	log.Info("creating new task queue", zap.Int("queue size", queueSize))
	return &TaskQueue{
		log:          log,
		lockedIssues: make(map[int]bool, queueSize),
		lockedPRs:    make(map[int]bool, queueSize),
		queue:        make(chan Task, queueSize),
	}
}

func (q *TaskQueue) PushComment(comment vc.Comment) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.lockedPRs[comment.PRNumber] {
		q.log.Info("skip adding comment to queue because PR is locked", zap.Int("PR number", comment.PRNumber))
		return
	}
	newTask := Task{
		TaskType: CommentTask,
		Comment:  comment,
	}
	q.lockedPRs[comment.PRNumber] = true
	q.queue <- newTask
}

func (q *TaskQueue) PushIssue(issue vc.Issue) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.lockedIssues[issue.Number] {
		q.log.Info("skip adding issue to queue because issue is locked", zap.Int("issue number", issue.Number))
		return
	}
	newTask := Task{
		TaskType: IssueTask,
		Issue:    issue,
	}
	q.lockedIssues[issue.Number] = true
	q.queue <- newTask
}

func (q *TaskQueue) ProcessAll(issueCb func(vc.Issue), commentCb func(vc.Comment)) {
	for len(q.queue) > 0 {
		q.ProcessNext(issueCb, commentCb)
	}
}

func (q *TaskQueue) ProcessNext(issueCb func(vc.Issue), commentCb func(vc.Comment)) {
	if len(q.queue) == 0 {
		q.log.Info("task queue empty; skipping process step")
		return
	}
	nextTask := <-q.queue
	switch nextTask.TaskType {
	case IssueTask:
		issueCb(nextTask.Issue)
		q.log.Info("finished processing issue", zap.Int("issue number", nextTask.Issue.Number))
		q.mu.Lock()
		delete(q.lockedIssues, nextTask.Issue.Number)
		q.mu.Unlock()
	case CommentTask:
		commentCb(nextTask.Comment)
		q.log.Info("finished processing comment", zap.Int("pr number", nextTask.Comment.PRNumber))
		q.mu.Lock()
		delete(q.lockedPRs, nextTask.Comment.PRNumber)
		q.mu.Unlock()
	}
}
