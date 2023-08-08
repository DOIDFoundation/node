//go:build sqlite

package core

func (a *API) GetBlockByMiner(param *GetBlockByMinerParam) *GetBlockByMinerData {
	miner := param.Miner
	limit := param.Limit
	page := param.Page
	totalRecords := a.chain.blockStore.MinerDb.CountBlockByMiner(miner, limit)
	totalPage := (totalRecords + limit - 1) / limit
	if page <= 0 || page > totalPage {
		return nil
	}
	data := a.chain.blockStore.MinerDb.QueryBlockByMiner(miner, limit, page)

	return &GetBlockByMinerData{Data: data, TotalPage: totalPage}
}
