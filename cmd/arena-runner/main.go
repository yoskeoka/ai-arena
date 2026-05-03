package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/yoskeoka/ai-arena/internal/games/echo"
	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
	"github.com/yoskeoka/ai-arena/internal/platform/runtime"
	"github.com/yoskeoka/ai-arena/internal/platform/session"
)

const defaultStderrLimitBytes = 4096

type playerSpec struct {
	PlayerID string
	Entry    string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	var (
		gameName         string
		gameVersion      string
		ruleset          string
		matchID          string
		stderrLimitBytes int
		playerArgs       multiFlag
	)

	fs := flag.NewFlagSet("arena-runner", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&gameName, "game", "", "game id")
	fs.StringVar(&gameVersion, "game-version", "", "game version")
	fs.StringVar(&ruleset, "ruleset", "", "game ruleset")
	fs.StringVar(&matchID, "match-id", "", "match id")
	fs.IntVar(&stderrLimitBytes, "stderr-limit-bytes", defaultStderrLimitBytes, "captured stderr bytes per player")
	fs.Var(&playerArgs, "player", "player_id=entry-path")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if gameName != echo.GameID {
		return fmt.Errorf("unsupported game %q", gameName)
	}
	if len(playerArgs) == 0 {
		return fmt.Errorf("at least one --player is required")
	}
	if gameVersion == "" {
		return fmt.Errorf("--game-version is required")
	}
	if ruleset == "" {
		return fmt.Errorf("--ruleset is required")
	}
	if matchID == "" {
		matchID = "match-" + uuid.NewString()
	}

	playersForGame, err := parsePlayersForGame(playerArgs)
	if err != nil {
		return err
	}

	master, err := newMaster(gameName, gameVersion, ruleset, playersForGame)
	if err != nil {
		return err
	}
	if meta := master.Metadata(); meta.GameVersion != gameVersion {
		return fmt.Errorf("selected game version %q does not match implementation version %q", gameVersion, meta.GameVersion)
	}

	players, sessions, err := loadPlayersAndSessions(master.Metadata(), playerArgs, stderrLimitBytes)
	if err != nil {
		return err
	}

	record, runErr := match.NewRunner(matchID, players, master, sessions).Run(context.Background())
	if err := json.NewEncoder(os.Stdout).Encode(record); err != nil {
		return err
	}
	if runErr != nil {
		fmt.Fprintln(os.Stderr, runErr)
	}
	return nil
}

func newMaster(gameName, gameVersion, ruleset string, players []game.Player) (game.Master, error) {
	switch gameName {
	case echo.GameID:
		return echo.New(echo.Config{
			GameVersion: gameVersion,
			Ruleset:     ruleset,
			Players:     players,
		})
	default:
		return nil, fmt.Errorf("unsupported game %q", gameName)
	}
}

func parsePlayersForGame(args []string) ([]game.Player, error) {
	players := make([]game.Player, 0, len(args))
	seenPlayerIDs := make(map[string]struct{}, len(args))
	for _, arg := range args {
		spec, err := parsePlayerSpec(arg)
		if err != nil {
			return nil, err
		}
		if _, exists := seenPlayerIDs[spec.PlayerID]; exists {
			return nil, fmt.Errorf("duplicate player_id %q", spec.PlayerID)
		}
		seenPlayerIDs[spec.PlayerID] = struct{}{}
		players = append(players, game.Player{
			PlayerID: spec.PlayerID,
			AIID:     spec.PlayerID,
		})
	}
	return players, nil
}

func loadPlayersAndSessions(meta catalog.GameMetadata, args []string, stderrLimitBytes int) ([]game.Player, map[string]match.PlayerSession, error) {
	players := make([]game.Player, 0, len(args))
	sessions := make(map[string]match.PlayerSession, len(args))
	seenPlayerIDs := make(map[string]struct{}, len(args))
	for _, arg := range args {
		spec, err := parsePlayerSpec(arg)
		if err != nil {
			return nil, nil, err
		}
		if _, exists := seenPlayerIDs[spec.PlayerID]; exists {
			closeSessions(sessions)
			return nil, nil, fmt.Errorf("duplicate player_id %q", spec.PlayerID)
		}
		seenPlayerIDs[spec.PlayerID] = struct{}{}
		loaded, err := loadEntry(meta, spec)
		if err != nil {
			closeSessions(sessions)
			return nil, nil, err
		}
		players = append(players, game.Player{
			PlayerID: spec.PlayerID,
			AIID:     loaded.AIID,
		})
		adapter, err := runtime.Start(context.Background(), runtime.Config{
			Command:          loaded.Command,
			Dir:              repoRoot(),
			StderrLimitBytes: stderrLimitBytes,
		})
		if err != nil {
			closeSessions(sessions)
			return nil, nil, err
		}
		sessions[spec.PlayerID] = session.New(adapter)
	}
	return players, sessions, nil
}

type loadedEntry struct {
	Command []string
	AIID    string
}

func loadEntry(matchMeta catalog.GameMetadata, spec playerSpec) (loadedEntry, error) {
	sidecarPath := spec.Entry + ".arena.json"
	if _, err := os.Stat(sidecarPath); err == nil {
		var manifest catalog.SidecarManifest
		if err := readJSON(sidecarPath, &manifest); err != nil {
			return loadedEntry{}, err
		}
		aiMeta := catalog.GameMetadata{
			GameID:         manifest.Protocol.GameID,
			GameVersion:    manifest.Protocol.GameVersion,
			RulesetVersion: manifest.Protocol.RulesetVersion,
			TurnMode:       matchMeta.TurnMode,
		}
		if err := catalog.Compatible(matchMeta, aiMeta); err != nil {
			return loadedEntry{}, fmt.Errorf("%s metadata incompatible: %w", spec.PlayerID, err)
		}
		if manifest.Protocol.Transport != "" && manifest.Protocol.Transport != "stdio-jsonrpc-ndjson" {
			return loadedEntry{}, fmt.Errorf("%s transport %q is unsupported", spec.PlayerID, manifest.Protocol.Transport)
		}
		if len(manifest.Runtime.Command) == 0 {
			return loadedEntry{}, fmt.Errorf("%s runtime.command is required", spec.PlayerID)
		}
		return loadedEntry{Command: manifest.Runtime.Command, AIID: manifest.AIID}, nil
	} else if err != nil && !os.IsNotExist(err) {
		return loadedEntry{}, err
	}

	if err := catalog.ValidateMetadata(matchMeta); err != nil {
		return loadedEntry{}, err
	}
	return loadedEntry{Command: []string{spec.Entry}, AIID: filepath.Base(spec.Entry)}, nil
}

func parsePlayerSpec(raw string) (playerSpec, error) {
	playerID, entry, ok := strings.Cut(raw, "=")
	if !ok || playerID == "" || entry == "" {
		return playerSpec{}, fmt.Errorf("invalid --player %q", raw)
	}
	return playerSpec{PlayerID: playerID, Entry: entry}, nil
}

func readJSON(path string, dest any) error {
	root, err := os.OpenRoot(repoRoot())
	if err != nil {
		return err
	}
	defer root.Close()

	data, err := root.ReadFile(strings.TrimPrefix(filepath.Clean(path), "./"))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func repoRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func closeSessions(sessions map[string]match.PlayerSession) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	for _, sess := range sessions {
		_ = sess.Close(ctx)
	}
}

type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}
