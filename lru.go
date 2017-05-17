package nicecache

import (
	"sync"
)

// lru - это слайс, где индекс - это логарифм от количества вызовов элемента кэша с округлением вниз.
// внутри живет мапа, из которой при необходимости будем удалять элементы из нее
// если надо освободить какое-то количество элементов. то идем по слайсу и ищем ближайший индекс, в котором lruElement.count
// не равен нули - это будет бакет с элементами с наименьшим использованием
// мьютекс нужен, чтобы при расширении слайса мы не потеряли данные
type lru struct {
	arr []lruElement // use
	sync.RWMutex
}

type lruElement struct {
	m map[int]frequencyValue //map[indexInSlice]frequencyValue
	sync.RWMutex
	count *int32
}

type frequencyValue int
