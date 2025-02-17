package object

import (
	"bytes"
	"fmt"
	"math/big"
	"slices"
	"strings"

	"github.com/nspcc-dev/neofs-sdk-go/client"
	"github.com/nspcc-dev/neofs-sdk-go/object"
)

// IsIntegerSearchOp reports whether given op matches integer attributes.
func IsIntegerSearchOp(op object.SearchMatchType) bool {
	return op == object.MatchNumGT || op == object.MatchNumGE || op == object.MatchNumLT || op == object.MatchNumLE
}

// TODO: docs.
func MergeSearchResults(lim uint16, withAttr, cmpInt bool, sets [][]client.SearchResultItem, mores []bool) ([]client.SearchResultItem, bool, error) {
	if lim == 0 || len(sets) == 0 {
		return nil, false, nil
	}
	if len(sets) == 1 {
		n := min(uint16(len(sets[0])), lim)
		return sets[0][:n], n < lim || slices.Contains(mores, true), nil
	}
	lim = calcMaxUniqueSearchResults(lim, sets)
	res := make([]client.SearchResultItem, 0, lim)
	var more bool
	var minInt, curInt *big.Int
	if cmpInt {
		minInt, curInt = new(big.Int), new(big.Int)
	}
	for minInd := -1; ; minInd = -1 {
		for i := range sets {
			if len(sets[i]) == 0 {
				continue
			}
			if minInd < 0 {
				minInd = i
				if cmpInt {
					if _, ok := minInt.SetString(sets[i][0].Attributes[0], 10); !ok {
						return nil, false, fmt.Errorf("non-int attribute in result #%d", i)
					}
				}
				continue
			}
			cmpID := bytes.Compare(sets[i][0].ID[:], sets[minInd][0].ID[:])
			if cmpID == 0 {
				continue
			}
			if withAttr {
				var cmpAttr int
				if cmpInt {
					if _, ok := curInt.SetString(sets[i][0].Attributes[0], 10); !ok {
						return nil, false, fmt.Errorf("non-int attribute in result #%d", i)
					}
					cmpAttr = curInt.Cmp(minInt)
				} else {
					cmpAttr = strings.Compare(sets[i][0].Attributes[0], sets[minInd][0].Attributes[0])
				}
				if cmpAttr != 0 {
					if cmpAttr < 0 {
						minInd = i
						if cmpInt {
							minInt, curInt = curInt, new(big.Int)
						}
					}
					continue
				}
			}
			if cmpID < 0 {
				minInd = i
				if cmpInt {
					minInt, curInt = curInt, new(big.Int)
				}
			}
		}
		if minInd < 0 {
			break
		}
		res = append(res, sets[minInd][0])
		if uint16(len(res)) == lim {
			if more = len(sets[minInd]) > 1 || slices.Contains(mores, true); !more {
			loop:
				for i := range sets {
					if i == minInd {
						continue
					}
					for j := range sets[i] {
						if more = sets[i][j].ID != sets[minInd][0].ID; more {
							break loop
						}
					}
				}
			}
			break
		}
		for i := range sets {
			if i == minInd {
				continue
			}
			for j := range sets[i] {
				if sets[i][j].ID == sets[minInd][0].ID {
					sets[i] = sets[i][j+1:]
					break
				}
			}
		}
		sets[minInd] = sets[minInd][1:]
	}
	return res, more, nil
}

func calcMaxUniqueSearchResults(lim uint16, sets [][]client.SearchResultItem) uint16 {
	n := uint16(len(sets[0]))
	if n >= lim {
		return lim
	}
	for i := 1; i < len(sets); i++ {
	nextItem:
		for j := range sets[i] {
			for k := range i {
				for l := range sets[k] {
					if sets[k][l].ID == sets[i][j].ID {
						continue nextItem
					}
				}
			}
			if n++; n == lim {
				return n
			}
		}
	}
	return n
}
