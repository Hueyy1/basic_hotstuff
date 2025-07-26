package service

import (
	"hxy352/src/proto/basichotstuffpb"
	"hxy352/src/proto/commonpb"
	"hxy352/src/types"
	"sort"
	"sync"
)

func msgPriority(msg *basichotstuffpb.Msg) int {
	switch msg.Type {
	case commonpb.MessageType_NewView:
		return 0
	case commonpb.MessageType_Prepare:
		return 1
	case commonpb.MessageType_PrepareVote:
		return 2
	case commonpb.MessageType_PreCommit:
		return 3
	case commonpb.MessageType_PreCommitVote:
		return 4
	case commonpb.MessageType_Commit:
		return 5
	case commonpb.MessageType_CommitVote:
		return 6
	case commonpb.MessageType_Decide:
		return 7
	default:
		return 100 // unknown or lowest
	}
}

type MessageQueueService struct {
	mu sync.Mutex

	queue map[types.View][]*basichotstuffpb.Msg
}

func NewMessageQueueService() *MessageQueueService {
	return &MessageQueueService{
		queue: make(map[types.View][]*basichotstuffpb.Msg),
	}
}

func (s *MessageQueueService) Put(msg *basichotstuffpb.Msg) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.queue[types.View(msg.GetView())] = append(s.queue[types.View(msg.GetView())], msg)
}

func (s *MessageQueueService) Sort(view types.View) {
	sort.Slice(s.queue[view], func(i, j int) bool {

		// View 相同，按类型优先级排序
		return msgPriority(s.queue[view][i]) < msgPriority(s.queue[view][j])
	})
}

func (s *MessageQueueService) Pop(view types.View) *basichotstuffpb.Msg {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.queue[view] == nil || len(s.queue[view]) == 0 {
		return nil
	}

	s.Sort(view)

	// clean
	delete(s.queue, view-100)

	last := s.queue[view][len(s.queue[view])-1]
	s.queue[view] = s.queue[view][:len(s.queue[view])-1]

	return last
}

func (s *MessageQueueService) Get(view types.View) []*basichotstuffpb.Msg {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.queue[view]
}

// --------

type Queue[T any] struct {
	items []T
}

func NewQueue[T any]() *Queue[T] {
	return &Queue[T]{}
}

func (q *Queue[T]) Enqueue(item T) {
	q.items = append(q.items, item)
}

func (q *Queue[T]) Dequeue() (T, bool) {
	if len(q.items) == 0 {
		var zero T
		return zero, false
	}
	item := q.items[0]
	q.items = q.items[1:]
	return item, true
}

func (q *Queue[T]) IsEmpty() bool {
	return len(q.items) == 0
}

func (q *Queue[T]) Len() int {
	return len(q.items)
}
