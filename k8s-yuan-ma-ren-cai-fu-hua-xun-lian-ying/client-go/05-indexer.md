---
description: Indexer
---

# 05-Indexer

Indexer 是一个索引缓存，用于缓存 Reflector 从 Kube-apiserver _`List/Watch`_ 到的资源对象，可以理解是一个带索引查询的内存型存储。



下面通过走读源码的形式来了解 Indexer 的原理，本篇基于 k8s.io/client-go v0.30.0 源码讲解

## Indexer interface&#x20;

首先我们打开代码目录：kubernetes/staging/src/k8s.io/client-go/tools/cache/index.go

Indexer 有两个功能，缓存和索引：

```go
// Indexer extends Store with multiple indices and restricts each
// accumulator to simply hold the current object (and be empty after
// Delete).
//
// There are three kinds of strings here:
//  1. a storage key, as defined in the Store interface,
//  2. a name of an index, and
//  3. an "indexed value", which is produced by an IndexFunc and
//     can be a field value or any other string computed from the object.
type Indexer interface {
	Store
	// Index returns the stored objects whose set of indexed values
	// intersects the set of indexed values of the given object, for
	// the named index
	Index(indexName string, obj interface{}) ([]interface{}, error)
	// IndexKeys returns the storage keys of the stored objects whose
	// set of indexed values for the named index includes the given
	// indexed value
	IndexKeys(indexName, indexedValue string) ([]string, error)
	// ListIndexFuncValues returns all the indexed values of the given index
	ListIndexFuncValues(indexName string) []string
	// ByIndex returns the stored objects whose set of indexed values
	// for the named index includes the given indexed value
	ByIndex(indexName, indexedValue string) ([]interface{}, error)
	// GetIndexers return the indexers
	GetIndexers() Indexers

	// AddIndexers adds more indexers to this store. This supports adding indexes after the store already has items.
	AddIndexers(newIndexers Indexers) error
}

```

### store

Store 接口实现了对 obj 的增删改查，具体实现 为 cache

```go
// Store is a generic object storage and processing interface.  A
// Store holds a map from string keys to accumulators, and has
// operations to add, update, and delete a given object to/from the
// accumulator currently associated with a given key.  A Store also
// knows how to extract the key from a given object, so many operations
// are given only the object.
//
// In the simplest Store implementations each accumulator is simply
// the last given object, or empty after Delete, and thus the Store's
// behavior is simple storage.
//
// Reflector knows how to watch a server and update a Store.  This
// package provides a variety of implementations of Store.
type Store interface {

	// Add adds the given object to the accumulator associated with the given object's key
	Add(obj interface{}) error

	// Update updates the given object in the accumulator associated with the given object's key
	Update(obj interface{}) error

	// Delete deletes the given object from the accumulator associated with the given object's key
	Delete(obj interface{}) error

	// List returns a list of all the currently non-empty accumulators
	List() []interface{}

	// ListKeys returns a list of all the keys currently associated with non-empty accumulators
	ListKeys() []string

	// Get returns the accumulator associated with the given object's key
	Get(obj interface{}) (item interface{}, exists bool, err error)

	// GetByKey returns the accumulator associated with the given key
	GetByKey(key string) (item interface{}, exists bool, err error)

	// Replace will delete the contents of the store, using instead the
	// given list. Store takes ownership of the list, you should not reference
	// it after calling this function.
	Replace([]interface{}, string) error

	// Resync is meaningless in the terms appearing here but has
	// meaning in some implementations that have non-trivial
	// additional behavior (e.g., DeltaFIFO).
	Resync() error
}
// KeyFunc knows how to make a key from an object. Implementations should be deterministic.
type KeyFunc func(obj interface{}) (string, error)

func MetaNamespaceKeyFunc(obj interface{}) (string, error) {
	if key, ok := obj.(ExplicitKey); ok {
		return string(key), nil
	}
	objName, err := ObjectToName(obj)
	if err != nil {
		return "", err
	}
	return objName.String(), nil
}
```

#### cache 实现

cache 实现了 store 接口，我们来看看 Add 方法，首先调动了 keyFunc 来获取对象的key，然后调用 c.cacheStorege.Add(key,ojb)来存储，这里将 keyFunc 和存储分开，是调用方更加灵活的存储对象，默认keyFunc是获取对象的 namespace/name

```go
// `*cache` implements Indexer in terms of a ThreadSafeStore and an
// associated KeyFunc.
type cache struct {
	// cacheStorage bears the burden of thread safety for the cache
	cacheStorage ThreadSafeStore
	// keyFunc is used to make the key for objects stored in and retrieved from items, and
	// should be deterministic.
	keyFunc KeyFunc
}


// Add inserts an item into the cache.
func (c *cache) Add(obj interface{}) error {
	key, err := c.keyFunc(obj)
	if err != nil {
		return KeyError{obj, err}
	}
	c.cacheStorage.Add(key, obj)
	return nil
}

// Update sets an item in the cache to its updated state.
func (c *cache) Update(obj interface{}) error {
	key, err := c.keyFunc(obj)
	if err != nil {
		return KeyError{obj, err}
	}
	c.cacheStorage.Update(key, obj)
	return nil
}
```

#### thredSaveMap&#x20;

每次添加对象到 items 中时，会同时调整索引

```go
// threadSafeMap implements ThreadSafeStore
type threadSafeMap struct {
	lock  sync.RWMutex
	items map[string]interface{}

	// index implements the indexing functionality
	index *storeIndex
}
func (c *threadSafeMap) Add(key string, obj interface{}) {
	c.Update(key, obj)
}

func (c *threadSafeMap) Update(key string, obj interface{}) {
	c.lock.Lock()
	defer c.lock.Unlock()
	oldObject := c.items[key]
	c.items[key] = obj
	c.index.updateIndices(oldObject, obj, key)
}
```

### Index

```go
// storeIndex implements the indexing functionality for Store interface
type storeIndex struct {
	// indexers maps a name to an IndexFunc
	indexers Indexers
	// indices maps a name to an Index
	indices Indices
}

// Index maps the indexed value to a set of keys in the store that match on that value
type Index map[string]sets.String

// Indexers maps a name to an IndexFunc
type Indexers map[string]IndexFunc

// Indices maps a name to an Index
type Indices map[string]Index

```

上面有几个比较难以理解的名词：_**Indexers**、**IndexFunc**、**Indices**、**Index**_，这四个名词都是关于索引功能的，下面通过例子来阐述。

#### Indexers 和 IndexFunc

Indexers 包含了所有**索引器**(索引分类)及其**索引器函数** IndexFunc，IndexFunc 为计算某个**索引键**下的所有**对象键**列表的方法；

Indexers：**索引器**

IndexFunc：**索引器函数**

```go

Indexers: {  
  "索引器1": 索引函数1,
  "索引器2": 索引函数2,
}

// 示例
Indexers: {  
  "namespace": MetaNamespaceIndexFunc,
  "nodeName": NodeNameIndexFunc,
}

// MetaNamespaceIndexFunc 获取对象的 namespace，该 namespace 作为索引键
// NodeNameIndexFunc 获取对象的 nodeName，该 nodeName 作为索引键


func MetaNamespaceIndexFunc(obj interface{}) ([]string, error) {
  meta, err := meta.Accessor(obj)
  if err != nil {
    return []string{""}, fmt.Errorf("object has no meta: %v", err)
  }
  return []string{meta.GetNamespace()}, nil
}

func NodeNameIndexFunc(obj interface{}) ([]string, error) {
  pod, ok := obj.(*v1.Pod)
  if !ok {
    return []string{""}, fmt.Errorf("object is not a pod)
  }
  return []string{pod.Spec.NodeName}, nil
}
```

#### **Indices、Index**

Indices 包含了所有**索引器**(索引分类)及其所有的索引数据 Index；而 Index 则包含了**索引键**以及索引键下的所有**对象键**的列表

```go

Indices: {
  # index
 "索引器1": {  
  "索引键1": ["对象键1", "对象键2"],  
  "索引键2": ["对象键3"],   
 },
 "索引器2": {  
  "索引键3": ["对象键1"],  
  "索引键4": ["对象键2", "对象键3"],  
 }
}
```

```go
// 示例：
pod1 := &v1.Pod {
    ObjectMeta: metav1.ObjectMeta {
        Name: "pod-1",
        Namespace: "default",
    },
    Spec: v1.PodSpec{
        NodeName: "node1",
    }
}

pod2 := &v1.Pod {
    ObjectMeta: metav1.ObjectMeta {
        Name: "pod-2",
        Namespace: "default",
    },
    Spec: v1.PodSpec{
        NodeName: "node2",
    }
}

pod3 := &v1.Pod {
    ObjectMeta: metav1.ObjectMeta {
        Name: "pod-3",
        Namespace: "kube-system",
    },
    Spec: v1.PodSpec{
        NodeName: "node2",
    }
}
Indices: {
 "namespace": {  
  "default": ["pod-1", "pod-2"],  
  "kube-system": ["pod-3"],   
 },
 "nodeName": {  
  "node1": ["pod-1"],  
  "node2": ["pod-2", "pod-3"],  
 }
}
```

<figure><img src="../../.gitbook/assets/image (3) (1) (1) (1) (1) (1) (1).png" alt=""><figcaption></figcaption></figure>

了解了底层数据接口，我们再回过头来看看其暴露给外部的方法：

```go
// Indexer extends Store with multiple indices and restricts each
// accumulator to simply hold the current object (and be empty after
// Delete).
//
// There are three kinds of strings here:
//  1. a storage key, as defined in the Store interface,
//  2. a name of an index, and
//  3. an "indexed value", which is produced by an IndexFunc and
//     can be a field value or any other string computed from the object.
type Indexer interface {
	Store
	// Index returns the stored objects whose set of indexed values
	// intersects the set of indexed values of the given object, for
	// the named index
	Index(indexName string, obj interface{}) ([]interface{}, error)
	// IndexKeys returns the storage keys of the stored objects whose
	// set of indexed values for the named index includes the given
	// indexed value
	IndexKeys(indexName, indexedValue string) ([]string, error)
	// ListIndexFuncValues returns all the indexed values of the given index
	ListIndexFuncValues(indexName string) []string
	// ByIndex returns the stored objects whose set of indexed values
	// for the named index includes the given indexed value
	ByIndex(indexName, indexedValue string) ([]interface{}, error)
	// GetIndexers return the indexers
	GetIndexers() Indexers

	// AddIndexers adds more indexers to this store. This supports adding indexes after the store already has items.
	AddIndexers(newIndexers Indexers) error
}
```

#### index 方法用于获取指定索引的所有对象

这个 Index 函数就是获取一个指定对象的索引键，然后把这个索引键下面的所有的对象全部获取到，比如我们要获取一个 Pod 所在命名空间下面的所有 Pod，如果更抽象一点，就是符合对&#x8C61;_&#x67D0;些特&#x5F81;_&#x7684;所有对象，而这个特征就是我们指定的索引键函数计算出来的。

```go
// Index returns a list of items that match the given object on the index function.
// Index is thread-safe so long as you treat all items as immutable.
func (c *threadSafeMap) Index(indexName string, obj interface{}) ([]interface{}, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	storeKeySet, err := c.index.getKeysFromIndex(indexName, obj)
	if err != nil {
		return nil, err
	}
	// 拿到了所有的对象键集合过后，循环拿到所有的对象集合
	list := make([]interface{}, 0, storeKeySet.Len())
	for storeKey := range storeKeySet {
		list = append(list, c.items[storeKey])
	}
	return list, nil
}

func (i *storeIndex) getKeysFromIndex(indexName string, obj interface{}) (sets.String, error) {
	// 获得索引器 indexName 的索引键计算函数
	indexFunc := i.indexers[indexName]
	if indexFunc == nil {
		return nil, fmt.Errorf("Index with name %s does not exist", indexName)
	}
	// 获取指定 obj 对象的索引键
	indexedValues, err := indexFunc(obj)
	if err != nil {
		return nil, err
	}
	// 获得索引器 indexName 的所有索引
	index := i.indices[indexName]

	var storeKeySet sets.String
	if len(indexedValues) == 1 {
		// In majority of cases, there is exactly one value matching.
		// Optimize the most common path - deduping is not needed here.
		 // 大多数情况下只有一个值匹配（默认获取的索引键就是对象的 namespace）
    		 // 直接拿到这个索引键的对象键集合
		storeKeySet = index[indexedValues[0]]
	} else {
		// Need to de-dupe the return list.
		// Since multiple keys are allowed, this can happen.
		 // 由于有多个索引键，则可能有重复的对象键出现，索引需要去重
		storeKeySet = sets.String{}
		for _, indexedValue := range indexedValues {
			for key := range index[indexedValue] {
				storeKeySet.Insert(key)
			}
		}
	}

	return storeKeySet, nil
}
```

#### ByIndex <a href="#byindex" id="byindex"></a>

然后接下来就是一个比较重要的 `ByIndex` 函数的实现, 可以很清楚地看到 `ByIndex` 函数和 Index 函数比较类似，但是更简单了，因为不需要通过索引函数计算索引键了，直接获取一个指定的索引键的全部资源对象。

```go
// ByIndex returns a list of the items whose indexed values in the given index include the given indexed value
func (c *threadSafeMap) ByIndex(indexName, indexedValue string) ([]interface{}, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	set, err := c.index.getKeysByIndex(indexName, indexedValue)
	if err != nil {
		return nil, err
	}
	list := make([]interface{}, 0, set.Len())
	for key := range set {
		list = append(list, c.items[key])
	}

	return list, nil
}
func (i *storeIndex) getKeysByIndex(indexName, indexedValue string) (sets.String, error) {
	 // 获得索引器 indexName 的索引键计算函数
	indexFunc := i.indexers[indexName]
	if indexFunc == nil {
		return nil, fmt.Errorf("Index with name %s does not exist", indexName)
	}
	// 获得索引器 indexName 的所有索引
	index := i.indices[indexName]
	return index[indexedValue], nil
}
```

#### indeKeys&#x20;

```go
// k8s.io/client-go/tools/cache/thread_safe_store.go

// IndexKeys 和上面的 ByIndex 几乎是一样的，只是这里是直接返回对象键列表
```

```go
// IndexKeys returns a list of the Store keys of the objects whose indexed values in the given index include the given indexed value.
// IndexKeys is thread-safe so long as you treat all items as immutable.
func (c *threadSafeMap) IndexKeys(indexName, indexedValue string) ([]string, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	set, err := c.index.getKeysByIndex(indexName, indexedValue)
	if err != nil {
		return nil, err
	}
	return set.List(), nil
}
```

这里我们就将 ThreadSafeMap 的实现进行了分析说明。整体来说比较方便，一个就是将对象数据存入到一个 map 中，然后就是维护索引，方便根据索引来查找到对应的对象。
