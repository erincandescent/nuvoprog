-- Copyright © 2019 Erin Shepherd
--
-- Licensed under the Apache License, Version 2.0 (the "License");
-- you may not use this file except in compliance with the License.
-- You may obtain a copy of the License at
--
--     http://www.apache.org/licenses/LICENSE-2.0
--
-- Unless required by applicable law or agreed to in writing, software
-- distributed under the License is distributed on an "AS IS" BASIS,
-- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
-- See the License for the specific language governing permissions and
-- limitations under the License.

-- Wireshark dissector for the NuLink 1 protocol.

local nl_proto = Proto("nulink1","NuLink 1 Protocol")

local cmd_names = {
	[0xFF] = "GET_VERSION",
	[0xA0] = "WRITE_FLASH (8051)",
	[0xA1] = "READ_FLASH?",
	[0xA2] = "SET_CONFIG",
	[0xA3] = "CHECK_ID",
	[0xA4] = "ERASE_FLASHCHIP",
	[0xB8] = "WRITE_REG",
	[0xB9] = "WRITE_RAM",
	[0xD1] = "MCU_STEP_RUN",
	[0xD2] = "MCU_STOP_RUN",
	[0xD3] = "MCU_FREE_RUN",
	[0xD8] = "CHECK_MCU_STOP",
	[0xE2] = "MCU_RESET",
}

local flash_spaces = {
	[0x00] = "APROM",
	[0x03] = "CONFIG",
}

local chip_type = {
	[0x321] = "NUC_CHIP_TYPE_M2351",
	[0x800] = "NUC_CHIP_TYPE_N76E003" -- This may be more generic but will need experimental determination
}

local mcu_reset_type = {
	[0] = "RESET_AUTO",
	[1] = "RESET_HW",
	[2] = "RESET_SYSRESETREQ",
	[3] = "RESET_VECTRESET",
	[4] = "RESET_FAST_RESCUE",
	[5] = "RESET_NONE_NULINK",
	[6] = "RESET_NONE2 (8051T1 family only)",
}

local reset_conn_type = {
	[0] = "CONNECT_NORMAL",
	[1] = "CONNECT_PRE_RESET",
	[2] = "CONNECT_UNDER_RESET",
	[3] = "CONNECT_NONE",
	[4] = "CONNECT_DISCONNECT",
	[5] = "CONNECT_ICP_MODE",
}

local reset_mode = {
	[0] = "Ext Mode"
}

local chip_id = {
	-- Note: IAP commands return
	-- Company ID Read: 0xDA
	-- Device ID Read:  0x3650
	[0xda3650] = "N76E003"
}

local cmds = {}

local f_seqno = ProtoField.uint8("nulink.seqno", "Sequence Number", base.DEC)
local f_len   = ProtoField.uint8("nulink.len", "Length", base.DEC)
local f_cmd   = ProtoField.uint32("nulink.cmd", "Command", base.HEX, cmd_names)
local f_body  = ProtoField.bytes("nulink.body", "Body")
local f_addr  = ProtoField.uint16("nulink.addr", "Address", base.HEX)
local f_space = ProtoField.uint8("nulink.flash_space", "Flash Space", base.HEX, flash_spaces)
local f_wrlen = ProtoField.uint8("nulink.wrlen", "Write Length", base.DEC)
local f_rdlen = ProtoField.uint8("nulink.rdlen", "Read Length", base.DEC)
local f_bytes = ProtoField.bytes("nulink.data", "Data")

local f_cfg_clock   = ProtoField.uint32("nulink.cfg.clk", "Debug Clock", base.DEC)
local f_cfg_chip    = ProtoField.uint32("nulink.cfg.chip", "Chip Type", base.HEX, chip_type)
local f_cfg_voltage = ProtoField.uint32("nulink.cfg.voltage", "Voltage", base.DEC)
local f_cfg_power_target = ProtoField.uint32("nulink.cfg.power_target", "Power Target", base.DEC)
local f_cfg_usb_func_e = ProtoField.uint32("nulink.cfg.usb_func_e", "USB_FUNC_E", base.DEC)

local f_reset_type      = ProtoField.uint32("nulink.mcu_reset.type",      "Reset Type",      base.HEX, mcu_reset_type)
local f_reset_conn_type = ProtoField.uint32("nulink.mcu_reset.conn_type", "Connection Type", base.HEX, reset_conn_type)
local f_reset_mode      = ProtoField.uint32("nulink.mcu_reset.mode",      "Mode",            base.HEX, reset_mode)

local f_chip_id = ProtoField.uint32("nulink.chip_id", "Chip ID", base.HEX, chip_id)

-- WRITE_FLASH
cmds[0xA0] = function(buf, pinfo, tree)
	tree:add(f_addr,  buf(0, 2), buf(0, 2):le_uint())
	tree:add(f_space, buf(2, 1))
	tree:add(f_wrlen, buf(4, 1))
	tree:add(f_bytes, buf(8, len))

	pinfo.cols.info:set(string.format("WRITE_FLASH 0x%04x %d bytes", buf(0, 2):le_uint(), buf(4,1):uint()))
end

-- READ_FLASH
cmds[0xA1] = function(buf, pinfo, tree)
	tree:add(f_addr,  buf(0, 2), buf(0, 2):le_uint())
	tree:add(f_space, buf(2, 1))
	tree:add(f_rdlen, buf(4, 1))

	pinfo.cols.info:set(string.format("READ_FLASH 0x%04x %d bytes", buf(0, 2):le_uint(), buf(4,1):uint()))
end

-- SET_CONFIG
cmds[0xA2] = function(buf, pinfo, tree)
	tree:add(f_cfg_clock,        buf(0,  4), buf(0,  4):le_uint())
	tree:add(f_cfg_chip,         buf(4,  4), buf(4,  4):le_uint())
	tree:add(f_cfg_voltage,      buf(8,  4), buf(8,  4):le_uint())
	tree:add(f_cfg_power_target, buf(12, 4), buf(12, 4):le_uint())
	tree:add(f_cfg_usb_func_e,   buf(16, 4), buf(16, 4):le_uint())
end

-- MCU_RESET
cmds[0xE2] = function(buf, pinfo, tree)
	tree:add(f_reset_type,        buf(0,  4), buf(0,  4):le_uint())
	tree:add(f_reset_conn_type,   buf(4,  4), buf(4,  4):le_uint())
	tree:add(f_reset_mode,        buf(8,  4), buf(8,  4):le_uint())
end

-- CHECK_ID
cmds[0xA3] = function(buf, pinfo, tree)
	tree:add(f_chip_id,           buf(0,  4), buf(0,  4):le_uint())
end

nl_proto.fields = {
	f_seqno,
	f_len,
	f_cmd,
	f_body,
	f_addr,
	f_wrlen,
	f_rdlen,
	f_bytes,
	f_space ,
	f_cfg_clock,
	f_cfg_chip,
	f_cfg_voltage,
	f_cfg_power_target,
	f_cfg_usb_func_e,
	f_reset_type,
	f_reset_conn_type,
	f_reset_mode,
	f_chip_id,
}

function nl_proto.dissector(buf,pinfo,tree)
    pinfo.cols.protocol = "NuLink"
	local subtree = tree:add(nl_proto, buf(0,64))

	local len = buf(1,1):uint()
	local cmd = buf(2,4):le_uint()
	local body_range = buf(6, len - 4)

    subtree:add(f_seqno, buf(0,1))
    subtree:add(f_len,   buf(1,1), len)
    subtree:add(f_cmd,   buf(2,4), cmd)
    local body_item = subtree:add(f_body, body_range)

    cmd = buf(2,1):uint()
    if cmd_names[cmd] ~= nil then
		pinfo.cols.info:set(cmd_names[cmd])
	else
		pinfo.cols.info:set(string.format("Unknown command %02x", cmd))
	end

	cmd_dis = cmds[cmd]
	if cmd_dis ~= nil and len > 4 then
		cmd_dis(body_range, pinfo, body_item)
	end
end

usb_bulk_table = DissectorTable.get("usb.bulk")
usb_bulk_table:add(0x0A, nl_proto)
