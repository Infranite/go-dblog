package types

import "strconv"

// Command is one decoded Redis command.
type Command struct {
	Name string
	Args []string
}

// CommandPlugin extends command parsing for product-specific Redis dialects.
type CommandPlugin interface {
	Name() string
	Match(Command) bool
	Apply(*Command) error
}

// Reverse returns a safe compensating command when the original command has
// enough information to undo the write without reading Redis state.
func (c Command) Reverse() (Command, bool) {
	switch c.Name {
	case CommandLPush:
		if len(c.Args) < 2 {
			return Command{}, false
		}
		return Command{Name: CommandLPop, Args: []string{c.Args[0], strconv.Itoa(len(c.Args) - 1)}}, true
	case CommandRPush:
		if len(c.Args) < 2 {
			return Command{}, false
		}
		return Command{Name: CommandRPop, Args: []string{c.Args[0], strconv.Itoa(len(c.Args) - 1)}}, true
	case CommandIncr:
		if len(c.Args) != 1 {
			return Command{}, false
		}
		return Command{Name: CommandDecr, Args: append([]string(nil), c.Args...)}, true
	case CommandDecr:
		if len(c.Args) != 1 {
			return Command{}, false
		}
		return Command{Name: CommandIncr, Args: append([]string(nil), c.Args...)}, true
	case CommandIncrBy:
		if len(c.Args) != 2 {
			return Command{}, false
		}
		return Command{Name: CommandDecrBy, Args: append([]string(nil), c.Args...)}, true
	case CommandDecrBy:
		if len(c.Args) != 2 {
			return Command{}, false
		}
		return Command{Name: CommandIncrBy, Args: append([]string(nil), c.Args...)}, true
	default:
		return Command{}, false
	}
}
