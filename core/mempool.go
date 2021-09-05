package core

type MemPoolTX struct {
	Tx  *Transaction
	Fee uint64
}

type MemPoolHeap []*MemPoolTX

func (mh MemPoolHeap) Len() int {
	return len(mh)
}

func (mh MemPoolHeap) Less(i, j int) bool {
	// the MemPool should be a max heap to make TX picking for candidate block easier
	return mh[i].Fee < mh[j].Fee
}

func (mh MemPoolHeap) Swap(i, j int) {
	mh[i], mh[j] = mh[j], mh[i]
}

func (mh *MemPoolHeap) Push(x interface{}) {
	*mh = append(*mh, x.(*MemPoolTX))
}

func (mh *MemPoolHeap) Pop() interface{} {
	x := (*mh)[len(*mh) - 1]
	*mh = (*mh)[0:len(*mh)-1]
	return x
}