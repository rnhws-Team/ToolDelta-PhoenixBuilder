// +build !is_tweak

package interfaces

import "phoenixbuilder/minecraft/protocol/packet"

type GameInterface interface {
	SendSettingsCommand(string, bool) error
	SendCommand(string) error
	SendWSCommand(string) error
	SendCommandWithResponse(string) (packet.CommandOutput, error)
	SendWSCommandWithResponse(string) (packet.CommandOutput, error)
	
	SetBlock([3]int32,string,string) error
	SetBlockForgetfully([3]int32,string,string) error
}