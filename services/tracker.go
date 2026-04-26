package services

import (
	"sync"
	"time"

	commonCtx "github.com/Gealber/common/context"
	"github.com/rs/zerolog/log"

	gossipDto "github.com/Gealber/gossip/dto"
	gossipSvc "github.com/Gealber/gossip/services"
	grpcClt "github.com/Gealber/yellowstone-tritonone/client"
	"github.com/Gealber/yellowstone-tritonone/proto"
)

const MAX_SLOTS_TRACK = 1500

const TRACKER_SVC_ID = "tracker_svc"

type TrackerService struct {
	commonCtx.DefaultService

	clt    *grpcClt.Client
	gossip *gossipSvc.GossipService

	track [MAX_SLOTS_TRACK]trackEntry
}

type trackEntry struct {
	mu sync.Mutex

	slot          uint64
	gossipTsMs    int64
	yellowTsMs    int64
	latencyLogged bool
}

func (svc *TrackerService) Id() string {
	return TRACKER_SVC_ID
}

func (svc *TrackerService) Configure(ctx *commonCtx.Context) error {
	svc.gossip = &gossipSvc.GossipService{}
	err := svc.gossip.Configure(ctx)
	if err != nil {
		return err
	}

	return svc.DefaultService.Configure(ctx)
}

func (svc *TrackerService) Start() error {
	var err error
	svc.clt, err = grpcClt.New(
		nil,
		nil,
		nil,
		false,
		true,
		svc.slotSub,
	)
	if err != nil {
		return err
	}

	voteCh := svc.gossip.Subscribe()
	go svc.readGossipSubCh(voteCh)

	go func() {
		err := svc.gossip.Start()
		if err != nil {
			log.Error().Err(err).Msg("ERROR STARTING GOSSIP SERVICE...")
		}
	}()

	return svc.clt.Run()
}

func (svc *TrackerService) Shutdown() {
	go svc.clt.Close()
	svc.gossip.Shutdown()
}

func (svc *TrackerService) readGossipSubCh(voteCh <-chan gossipDto.Vote) {
	for vote := range voteCh {
		timestampMs := time.Now().UnixMilli()
		latency, shouldLog := svc.trackGossip(vote.Slot, timestampMs)
		if !shouldLog {
			continue
		}

		log.Info().
			Uint64("slot", vote.Slot).
			Str("latency", latency.String()).
			Msg("SLOT LATENCY TRACKED")
	}
}

func (svc *TrackerService) slotSub(resp *proto.SubscribeUpdate) {
	slotUpd := resp.GetSlot()
	if slotUpd != nil {
		timestampMs := time.Now().UnixMilli()
		latency, shouldLog := svc.trackYellowstone(slotUpd.GetSlot(), timestampMs)
		if !shouldLog {
			return
		}

		log.Info().
			Uint64("slot", slotUpd.Slot).
			Str("latency", latency.String()).
			Msg("SLOT LATENCY TRACKED")
	}
}

func (svc *TrackerService) slotIdx(slot uint64) int {
	return int(slot % MAX_SLOTS_TRACK)
}

func (svc *TrackerService) trackGossip(slot uint64, timestampMs int64) (time.Duration, bool) {
	return svc.trackSlot(slot, timestampMs, true)
}

func (svc *TrackerService) trackYellowstone(slot uint64, timestampMs int64) (time.Duration, bool) {
	return svc.trackSlot(slot, timestampMs, false)
}

func (svc *TrackerService) trackSlot(slot uint64, timestampMs int64, fromGossip bool) (time.Duration, bool) {
	entry := &svc.track[svc.slotIdx(slot)]

	entry.mu.Lock()
	defer entry.mu.Unlock()

	if entry.slot != slot {
		entry.slot = slot
		entry.gossipTsMs = 0
		entry.yellowTsMs = 0
		entry.latencyLogged = false
	}

	if entry.latencyLogged {
		return 0, false
	}

	if fromGossip {
		if entry.gossipTsMs == 0 || timestampMs < entry.gossipTsMs {
			entry.gossipTsMs = timestampMs
		}
	} else {
		if entry.yellowTsMs == 0 || timestampMs < entry.yellowTsMs {
			entry.yellowTsMs = timestampMs
		}
	}

	if entry.gossipTsMs == 0 || entry.yellowTsMs == 0 {
		return 0, false
	}

	entry.latencyLogged = true
	return time.Duration(entry.yellowTsMs-entry.gossipTsMs) * time.Millisecond, true
}
