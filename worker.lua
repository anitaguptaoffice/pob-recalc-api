--
-- worker.lua — Long-lived LuaJIT worker for POB recalculation
--
-- This process stays alive after HeadlessWrapper initialization (~95% of startup cost).
-- It reads requests from stdin and writes results to stdout using a simple length-prefixed protocol.
--
-- Protocol (both directions):
--   REQUEST:  "RECALC <input_len>\n" followed by <input_len> bytes of XML
--   RESPONSE: "OK <output_len>\n" followed by <output_len> bytes of recalculated XML
--         OR: "ERR <msg_len>\n" followed by <msg_len> bytes of error message
--   READY:    "READY\n" — sent once after initialization is complete
--   PING/PONG: "PING\n" → "PONG\n" — health check
--   QUIT:     "QUIT\n" — graceful shutdown
--

-- CRITICAL: Redirect all print/ConPrintf to stderr BEFORE loading HeadlessWrapper.
-- stdout is reserved exclusively for the worker protocol.
local _real_stdout = io.stdout
local function stderrPrint(...)
	local args = {...}
	for i = 1, select("#", ...) do
		if i > 1 then io.stderr:write("\t") end
		io.stderr:write(tostring(args[i]))
	end
	io.stderr:write("\n")
end
print = stderrPrint

-- Override io.stdout temporarily during init to catch any stray writes.
-- We'll restore it after HeadlessWrapper is done.
-- (Some POB code may call print() or io.write() during init)

-- Patch missing stubs that HeadlessWrapper doesn't define but Launch.lua
-- may call before Common.lua is loaded (e.g. DrawPopup during OnInit error)
function GetVirtualScreenSize()
	return 1920, 1080
end

io.stderr:write("[worker] Initializing HeadlessWrapper...\n")

-- Load the headless wrapper — this is the expensive part (~2-5 seconds)
dofile("HeadlessWrapper.lua")

-- Also override ConPrintf in case it was redefined during init
ConPrintf = function(fmt, ...)
	io.stderr:write(string.format(fmt, ...) .. "\n")
end

-- Restore real stdout for protocol communication
io.stdout = _real_stdout

-- Run a few extra OnFrame calls to ensure item DBs are fully loaded
for i = 1, 20 do
	runCallback("OnFrame")
end

io.stderr:write("[worker] Initialization complete. Entering request loop.\n")

-- Signal readiness to the parent process
io.stdout:write("READY\n")
io.stdout:flush()

-- Helper: read exactly n bytes from stdin
local function readExact(n)
	local chunks = {}
	local remaining = n
	while remaining > 0 do
		local chunk = io.stdin:read(remaining)
		if not chunk then
			return nil, "unexpected EOF"
		end
		table.insert(chunks, chunk)
		remaining = remaining - #chunk
	end
	return table.concat(chunks)
end

-- Helper: send a response
local function sendOK(data)
	local header = string.format("OK %d\n", #data)
	io.stdout:write(header)
	io.stdout:write(data)
	io.stdout:flush()
end

local function sendErr(msg)
	local header = string.format("ERR %d\n", #msg)
	io.stdout:write(header)
	io.stdout:write(msg)
	io.stdout:flush()
end

-- Helper: minimal JSON encoder for flat and nested tables of numbers/strings
local function jsonEncode(val)
	if type(val) == "number" then
		-- Use integer format for whole numbers, otherwise float
		if val == math.floor(val) and val > -1e15 and val < 1e15 then
			return string.format("%.0f", val)
		end
		return string.format("%.6g", val)
	elseif type(val) == "string" then
		-- Escape special characters
		local escaped = val:gsub('\\', '\\\\'):gsub('"', '\\"'):gsub('\n', '\\n'):gsub('\r', '\\r'):gsub('\t', '\\t')
		return '"' .. escaped .. '"'
	elseif type(val) == "boolean" then
		return val and "true" or "false"
	elseif type(val) == "nil" then
		return "null"
	elseif type(val) == "table" then
		-- Check if it's an array (sequential integer keys starting from 1)
		local isArray = true
		local n = 0
		for k, _ in pairs(val) do
			n = n + 1
			if type(k) ~= "number" or k ~= n then
				isArray = false
				break
			end
		end
		if isArray and n > 0 then
			local parts = {}
			for i = 1, n do
				table.insert(parts, jsonEncode(val[i]))
			end
			return "[" .. table.concat(parts, ",") .. "]"
		else
			-- Object
			local parts = {}
			for k, v in pairs(val) do
				table.insert(parts, jsonEncode(tostring(k)) .. ":" .. jsonEncode(v))
			end
			return "{" .. table.concat(parts, ",") .. "}"
		end
	end
	return "null"
end

-- Helper: collect build stats from calcsTab.mainOutput
local function collectStats()
	local stats = {}
	if build and build.calcsTab and build.calcsTab.mainOutput then
		local out = build.calcsTab.mainOutput
		-- Core DPS stats
		stats.TotalDPS = out.TotalDPS or 0
		stats.CombinedDPS = out.CombinedDPS or 0
		stats.TotalDot = out.TotalDot or 0
		stats.BleedDPS = out.BleedDPS or 0
		stats.IgniteDPS = out.IgniteDPS or 0
		stats.PoisonDPS = out.PoisonDPS or 0
		stats.TotalDPS = out.TotalDPS or 0
		stats.AverageDamage = out.AverageDamage or 0
		stats.Speed = out.Speed or 0
		stats.CritChance = out.CritChance or 0
		stats.CritMultiplier = out.CritMultiplier or 0
		-- Defence stats
		stats.Life = out.Life or 0
		stats.LifeRegen = out.LifeRegen or 0
		stats.LifeUnreserved = out.LifeUnreserved or 0
		stats.EnergyShield = out.EnergyShield or 0
		stats.EnergyShieldRegen = out.EnergyShieldRegen or 0
		stats.Mana = out.Mana or 0
		stats.ManaUnreserved = out.ManaUnreserved or 0
		stats.ManaRegen = out.ManaRegen or 0
		stats.Armour = out.Armour or 0
		stats.Evasion = out.Evasion or 0
		stats.Ward = out.Ward or 0
		-- Resistances
		stats.FireResist = out.FireResist or 0
		stats.ColdResist = out.ColdResist or 0
		stats.LightningResist = out.LightningResist or 0
		stats.ChaosResist = out.ChaosResist or 0
		-- Block
		stats.BlockChance = out.BlockChance or 0
		stats.SpellBlockChance = out.SpellBlockChance or 0
	end
	return stats
end

-- Helper: generate recalculated XML by updating the <Build> section in the original XML
-- with new PlayerStat values from the recalculation, preserving all other sections
-- (Tree, Items, Skills, Config, etc.) exactly as they were in the input.
local function generateRecalcXML(originalXML)
	-- Parse the original XML into a DOM
	local dbXML, errMsg = common.xml.ParseXML(originalXML)
	if errMsg or not dbXML or not dbXML[1] then
		return nil, "Failed to parse original XML for recalc output: " .. tostring(errMsg)
	end

	-- Generate the new <Build> section from POB's recalculated state
	local newBuildNode = { elem = "Build" }
	build:Save(newBuildNode)

	-- Replace the <Build> node in the original XML DOM
	local replaced = false
	for i, node in ipairs(dbXML[1]) do
		if type(node) == "table" and node.elem == "Build" then
			dbXML[1][i] = newBuildNode
			replaced = true
			break
		end
	end
	if not replaced then
		-- If no Build node found, insert at the beginning
		table.insert(dbXML[1], 1, newBuildNode)
	end

	-- Compose back to XML text (dbXML[1] is the root <PathOfBuilding> node;
	-- ComposeXML expects a single node with .elem, not the wrapper array from ParseXML)
	local xmlText, composeErr = common.xml.ComposeXML(dbXML[1])
	if not xmlText then
		return nil, "Failed to compose recalc XML: " .. tostring(composeErr)
	end
	return xmlText
end

-- Main request loop
while true do
	local line = io.stdin:read("*l")
	if not line then
		io.stderr:write("[worker] stdin closed, exiting.\n")
		break
	end

	if line == "QUIT" then
		io.stderr:write("[worker] Received QUIT, exiting.\n")
		break
	end

	if line == "PING" then
		io.stdout:write("PONG\n")
		io.stdout:flush()
		goto continue
	end

	-- Parse REPLACE_ITEM command: "REPLACE_ITEM <xml_len> <slot_len> <item_len>"
	do
		local xmlLen, slotLen, itemLen = line:match("^REPLACE_ITEM (%d+) (%d+) (%d+)$")
		if xmlLen then
			xmlLen = tonumber(xmlLen)
			slotLen = tonumber(slotLen)
			itemLen = tonumber(itemLen)

			-- Read the three payloads
			local xmlText, err1 = readExact(xmlLen)
			if not xmlText then
				sendErr("failed to read xml: " .. (err1 or "unknown"))
				goto continue
			end

			local slotName, err2 = readExact(slotLen)
			if not slotName then
				sendErr("failed to read slot: " .. (err2 or "unknown"))
				goto continue
			end

			local itemText, err3 = readExact(itemLen)
			if not itemText then
				sendErr("failed to read item: " .. (err3 or "unknown"))
				goto continue
			end

			io.stderr:write(string.format("[worker] REPLACE_ITEM: xml=%d bytes, slot=%q, item=%d bytes\n",
				xmlLen, slotName, itemLen))

			-- Step 1: Load build and collect "before" stats
			local ok, errMsg = pcall(function()
				loadBuildFromXML(xmlText, "replace_item_before")
			end)
			if not ok then
				sendErr("loadBuildFromXML failed: " .. tostring(errMsg))
				goto continue
			end
			if not build or not build.calcsTab then
				sendErr("build or calcsTab not available after loading")
				goto continue
			end

			-- Ensure calculation is complete (buildFlag may be deferred)
			if build.buildFlag then
				for i = 1, 10 do
					runCallback("OnFrame")
					if not build.buildFlag then break end
				end
			end
			if not build.calcsTab.mainEnv then
				local buildOk, buildErr = pcall(function()
					build.calcsTab:BuildOutput()
				end)
				if not buildOk or not build.calcsTab.mainEnv then
					sendErr("BuildOutput failed for this build: " .. tostring(buildErr or "mainEnv is nil"))
					goto continue
				end
			end

			local beforeStats = collectStats()

			-- Step 2: Find the target slot and replace the item
			local itemsTab = build.itemsTab
			if not itemsTab then
				sendErr("build.itemsTab not available")
				goto continue
			end

			-- Find existing slot
			local targetSlot = nil
			for _, slot in pairs(itemsTab.slots) do
				if slot.slotName == slotName then
					targetSlot = slot
					break
				end
			end

			if not targetSlot then
				-- Build a list of available slots for the error message
				local available = {}
				for _, slot in pairs(itemsTab.slots) do
					if slot.slotName then
						table.insert(available, slot.slotName)
					end
				end
				table.sort(available)
				sendErr("slot not found: " .. slotName .. ". Available: " .. table.concat(available, ", "))
				goto continue
			end

			-- Parse the new item using POB's item parser
			local newItem = new("Item")
			newItem:ParseRaw(itemText)
			if newItem.base == nil then
				sendErr("failed to parse item text — invalid or unrecognized base type")
				goto continue
			end

			-- Add the new item to the build's item list
			local newItemId = #build.itemsTab.items + 1
			newItem.id = newItemId
			build.itemsTab.items[newItemId] = newItem

			-- Record old item ID for logging
			local oldItemId = targetSlot.selItemId or 0

			-- Assign the new item to the slot
			targetSlot.selItemId = newItemId

			-- Trigger full recalculation
			build.itemsTab:PopulateSlots()
			build.itemsTab:AddUndoState()
			build.buildFlag = true
			runCallback("OnFrame")

			local afterStats = collectStats()

			-- Step 3: Compute diffs
			local diff = {}
			for k, v in pairs(afterStats) do
				diff[k] = v - (beforeStats[k] or 0)
			end

			-- Build result
			local result = {
				slot = slotName,
				old_item_id = oldItemId,
				new_item_id = newItemId,
				before = beforeStats,
				after = afterStats,
				diff = diff,
			}

			local resultJSON = jsonEncode(result)

			io.stderr:write(string.format("[worker] REPLACE_ITEM complete: TotalDPS %.0f -> %.0f (%+.0f)\n",
				beforeStats.TotalDPS or 0, afterStats.TotalDPS or 0, diff.TotalDPS or 0))

			sendOK(resultJSON)
			goto continue
		end
	end

	-- Parse GENERATE_WEIGHTS command: "GENERATE_WEIGHTS <xml_len> <slot_len> <options_len>"
	do
		local xmlLen, slotLen, optsLen = line:match("^GENERATE_WEIGHTS (%d+) (%d+) (%d+)$")
		if xmlLen then
			xmlLen = tonumber(xmlLen)
			slotLen = tonumber(slotLen)
			optsLen = tonumber(optsLen)

			local xmlText, err1 = readExact(xmlLen)
			if not xmlText then
				sendErr("failed to read xml: " .. (err1 or "unknown"))
				goto continue
			end

			local slotName, err2 = readExact(slotLen)
			if not slotName then
				sendErr("failed to read slot: " .. (err2 or "unknown"))
				goto continue
			end

			local optsText, err3 = readExact(optsLen)
			if not optsText then
				sendErr("failed to read options: " .. (err3 or "unknown"))
				goto continue
			end

			io.stderr:write(string.format("[worker] GENERATE_WEIGHTS: xml=%d bytes, slot=%q, opts=%d bytes\n",
				xmlLen, slotName, optsLen))

			-- Parse options JSON (simple key extraction)
			-- Expected: {"stat_weights":[{"stat":"FullDPS","weightMult":1.0},...],"include_corrupted":false,...}
			local function parseSimpleJSON(s)
				-- Minimal JSON parser for our specific needs
				local result = {}
				-- Extract stat_weights array
				local swStr = s:match('"stat_weights"%s*:%s*(%[.-%])')
				if swStr then
					result.statWeights = {}
					for stat, wm in swStr:gmatch('"stat"%s*:%s*"([^"]+)"%s*,%s*"weightMult"%s*:%s*([%d%.%-]+)') do
						table.insert(result.statWeights, { stat = stat, weightMult = tonumber(wm) })
					end
				end
				-- Extract boolean flags
				for _, key in ipairs({"include_corrupted", "include_eldritch", "include_scourge", "include_synthesis", "include_talisman"}) do
					local val = s:match('"' .. key .. '"%s*:%s*(%a+)')
					if val then result[key] = (val == "true") end
				end
				return result
			end

			local opts = parseSimpleJSON(optsText)

			-- Default stat weights if none provided
			if not opts.statWeights or #opts.statWeights == 0 then
				opts.statWeights = {
					{ stat = "FullDPS", weightMult = 1.0 },
					{ stat = "TotalEHP", weightMult = 0.5 },
				}
			end

			-- Step 1: Load build
			local ok, errMsg = pcall(function()
				loadBuildFromXML(xmlText, "generate_weights")
			end)
			if not ok then
				sendErr("loadBuildFromXML failed: " .. tostring(errMsg))
				goto continue
			end
			if not build or not build.calcsTab then
				sendErr("build or calcsTab not available after loading")
				goto continue
			end

			-- Ensure calculation is complete (buildFlag may be deferred)
			if build.buildFlag then
				for i = 1, 10 do
					runCallback("OnFrame")
					if not build.buildFlag then break end
				end
			end
			if not build.calcsTab.mainEnv then
				local buildOk, buildErr = pcall(function()
					build.calcsTab:BuildOutput()
				end)
				if not buildOk or not build.calcsTab.mainEnv then
					sendErr("BuildOutput failed for this build: " .. tostring(buildErr or "mainEnv is nil"))
					goto continue
				end
			end

			-- Step 2: Determine slot and item category
			local itemsTab = build.itemsTab
			if not itemsTab then
				sendErr("build.itemsTab not available")
				goto continue
			end

			local targetSlot = nil
			for _, slot in pairs(itemsTab.slots) do
				if slot.slotName == slotName then
					targetSlot = slot
					break
				end
			end

			if not targetSlot then
				local available = {}
				for _, slot in pairs(itemsTab.slots) do
					if slot.slotName then
						table.insert(available, slot.slotName)
					end
				end
				table.sort(available)
				sendErr("slot not found: " .. slotName .. ". Available: " .. table.concat(available, ", "))
				goto continue
			end

			-- Determine item category based on slot and existing item
			local existingItem = targetSlot.selItemId and itemsTab.items[targetSlot.selItemId]
			local testItemType = existingItem and existingItem.baseName or "Unset Amulet"
			local itemCategory

			if slotName:find("^Weapon %d") then
				if existingItem then
					local t = existingItem.type
					if t == "Shield" then itemCategory = "Shield"
					elseif t == "Quiver" then itemCategory = "Quiver"
					elseif t == "Bow" then itemCategory = "Bow"
					elseif t == "Staff" then itemCategory = "Staff"
					elseif t == "Two Handed Sword" then itemCategory = "2HSword"
					elseif t == "Two Handed Axe" then itemCategory = "2HAxe"
					elseif t == "Two Handed Mace" then itemCategory = "2HMace"
					elseif t == "One Handed Sword" then itemCategory = "1HSword"
					elseif t == "One Handed Axe" then itemCategory = "1HAxe"
					elseif t == "One Handed Mace" or t == "Sceptre" then itemCategory = "1HMace"
					elseif t == "Wand" then itemCategory = "Wand"
					elseif t == "Dagger" then itemCategory = "Dagger"
					elseif t == "Claw" then itemCategory = "Claw"
					elseif t:find("Two Handed") then itemCategory = "2HWeapon"
					elseif t:find("One Handed") then itemCategory = "1HWeapon"
					else
						sendErr("unsupported weapon type: " .. t)
						goto continue
					end
				else
					itemCategory = "1HWeapon"
				end
			elseif slotName == "Body Armour" then itemCategory = "Chest"
			elseif slotName == "Helmet" then itemCategory = "Helmet"
			elseif slotName == "Gloves" then itemCategory = "Gloves"
			elseif slotName == "Boots" then itemCategory = "Boots"
			elseif slotName == "Amulet" then itemCategory = "Amulet"
			elseif slotName == "Ring 1" or slotName == "Ring 2" or slotName == "Ring 3" then itemCategory = "Ring"
			elseif slotName == "Belt" then itemCategory = "Belt"
			elseif slotName:find("Abyssal") then itemCategory = "AbyssJewel"
			elseif slotName:find("Jewel") then itemCategory = "BaseJewel"
			elseif slotName:find("Flask") then itemCategory = "Flask"
			else
				sendErr("unsupported slot for weight generation: " .. slotName)
				goto continue
			end

			io.stderr:write(string.format("[worker] GENERATE_WEIGHTS: itemCategory=%q, testItemType=%q\n",
				itemCategory, testItemType))

			-- Step 3: Load QueryMods data
			local modData
			local modDataOk, modDataErr = pcall(function()
				modData = LoadModule("Data/QueryMods.lua")
			end)
			if not modDataOk or not modData then
				sendErr("failed to load QueryMods.lua: " .. tostring(modDataErr))
				goto continue
			end

			io.stderr:write("[worker] GENERATE_WEIGHTS: QueryMods loaded\n")

			-- Step 4: Create blank test item and get calculator
			local testItem = new("Item", "Rarity: RARE\nStat Tester\n" .. testItemType)

			local calcFunc, baseOutput = build.calcsTab:GetMiscCalculator()

			-- Calculate baseline with empty test item
			local baseItemOutput = calcFunc({ repSlotName = targetSlot.slotName, repItem = testItem })

			-- WeightedRatioOutputs — ported from TradeQueryGenerator.lua
			local function weightedRatioOutputs(baseOut, newOut, statWeights)
				local meanStatDiff = 0
				local maxStatIncrease = data and data.misc and data.misc.maxStatIncrease or 1000
				local function ratioModSums(...)
					local baseModSum = 0
					local newModSum = 0
					for _, mod in ipairs({...}) do
						baseModSum = baseModSum + (baseOut[mod] or 0)
						newModSum = newModSum + (newOut[mod] or 0)
					end
					if baseModSum == math.huge then
						return 0
					else
						if newModSum == math.huge then
							return maxStatIncrease
						else
							return math.min(newModSum / ((baseModSum ~= 0) and baseModSum or 1), maxStatIncrease)
						end
					end
				end
				for _, statTable in ipairs(statWeights) do
					if statTable.stat == "FullDPS" and not (baseOut["FullDPS"] and newOut["FullDPS"]) then
						meanStatDiff = meanStatDiff + ratioModSums("TotalDPS", "TotalDotDPS", "CombinedDPS") * statTable.weightMult
					end
					meanStatDiff = meanStatDiff + ratioModSums(statTable.stat) * statTable.weightMult
				end
				return meanStatDiff
			end

			local baseStatValue = weightedRatioOutputs(baseOutput, baseItemOutput, opts.statWeights) * 1000

			-- Step 5: Test each mod and calculate weights
			local modWeights = {}
			local alreadyWeighted = {}
			local modsTested = 0

			local function generateModWeights(modsToTest)
				if not modsToTest then return end
				for _, entry in pairs(modsToTest) do
					if entry[itemCategory] ~= nil then
						if alreadyWeighted[entry.tradeMod.id] then
							goto skip
						end

						local modValue = math.ceil((entry[itemCategory].max - entry[itemCategory].min) * 0.5 + entry[itemCategory].min)
						if modValue == 0 then modValue = 1 end
						local modValueStr = (entry.sign and entry.sign or "") .. tostring(modValue)

						local modLine
						if modValue == 1 and entry.specialCaseData and entry.specialCaseData.overrideModLineSingular then
							modLine = entry.specialCaseData.overrideModLineSingular
						elseif entry.specialCaseData and entry.specialCaseData.overrideModLine then
							modLine = entry.specialCaseData.overrideModLine
						else
							modLine = entry.tradeMod.text
						end
						modLine = modLine:gsub("#", modValueStr)

						testItem.explicitModLines[1] = { line = modLine, custom = true }
						testItem:BuildAndParseRaw()

						local output = calcFunc({ repSlotName = targetSlot.slotName, repItem = testItem })
						local meanStatDiff = weightedRatioOutputs(baseOutput, output, opts.statWeights) * 1000 - baseStatValue
						if meanStatDiff > 0.01 then
							table.insert(modWeights, {
								tradeModId = entry.tradeMod.id,
								weight = meanStatDiff / modValue,
								meanStatDiff = meanStatDiff,
								modText = entry.tradeMod.text,
								modType = entry.tradeMod.type or "explicit",
								testValue = modValue,
								invert = entry.sign == "-" and true or false,
							})
						end
						alreadyWeighted[entry.tradeMod.id] = true
						modsTested = modsTested + 1
					end
					::skip::
				end
			end

			-- Run weight generation for each mod category
			generateModWeights(modData["Explicit"])
			generateModWeights(modData["Implicit"])
			if opts.include_corrupted then
				generateModWeights(modData["Corrupted"])
			end
			if opts.include_scourge then
				generateModWeights(modData["Scourge"])
			end
			if opts.include_eldritch then
				generateModWeights(modData["Eater"])
				generateModWeights(modData["Exarch"])
			end
			if opts.include_synthesis then
				generateModWeights(modData["Synthesis"])
			end

			io.stderr:write(string.format("[worker] GENERATE_WEIGHTS: tested %d mods, found %d with positive weight\n",
				modsTested, #modWeights))

			-- Step 6: Sort by meanStatDiff descending
			table.sort(modWeights, function(a, b)
				if a.meanStatDiff == b.meanStatDiff then
					return math.abs(a.weight) > math.abs(b.weight)
				end
				return a.meanStatDiff > b.meanStatDiff
			end)

			-- Step 7: Calculate current item's stat diff for reference
			local currentStatDiff = 0
			if existingItem then
				testItem.explicitModLines = {}
				for _, modLine in ipairs(existingItem.explicitModLines or {}) do
					table.insert(testItem.explicitModLines, modLine)
				end
				for _, modLine in ipairs(existingItem.implicitModLines or {}) do
					table.insert(testItem.explicitModLines, modLine)
				end
				testItem:BuildAndParseRaw()
				local origOutput = calcFunc({ repSlotName = targetSlot.slotName, repItem = testItem })
				currentStatDiff = weightedRatioOutputs(baseOutput, origOutput, opts.statWeights) * 1000 - baseStatValue
			end

			-- Step 8: Build result JSON
			-- Use jsonEncode for the array entries, build top-level manually for control
			local resultParts = {}
			table.insert(resultParts, '{"slot":' .. jsonEncode(slotName))
			table.insert(resultParts, ',"item_category":' .. jsonEncode(itemCategory))
			table.insert(resultParts, ',"current_item":' .. jsonEncode(existingItem and existingItem.name or ""))
			table.insert(resultParts, ',"current_stat_diff":' .. jsonEncode(currentStatDiff))
			table.insert(resultParts, ',"mods_tested":' .. jsonEncode(modsTested))
			table.insert(resultParts, ',"stat_weights":' .. jsonEncode(opts.statWeights))

			-- Encode mod_weights array
			local mwParts = {}
			for i, mw in ipairs(modWeights) do
				table.insert(mwParts, jsonEncode({
					trade_mod_id = mw.tradeModId,
					weight = mw.weight,
					mean_stat_diff = mw.meanStatDiff,
					mod_text = mw.modText,
					mod_type = mw.modType,
					test_value = mw.testValue,
					invert = mw.invert,
				}))
			end
			table.insert(resultParts, ',"mod_weights":[' .. table.concat(mwParts, ",") .. "]")
			table.insert(resultParts, "}")

			local resultJSON = table.concat(resultParts)

			io.stderr:write(string.format("[worker] GENERATE_WEIGHTS complete: %d weights, top weight=%s\n",
				#modWeights, #modWeights > 0 and modWeights[1].modText or "none"))

			sendOK(resultJSON)
			goto continue
		end
	end

	-- Parse FIND_BEST_ANOINT command: "FIND_BEST_ANOINT <xml_len> <opts_len>"
	do
		local xmlLen, optsLen = line:match("^FIND_BEST_ANOINT (%d+) (%d+)$")
		if xmlLen then
			xmlLen = tonumber(xmlLen)
			optsLen = tonumber(optsLen)

			local xmlText, err1 = readExact(xmlLen)
			if not xmlText then
				sendErr("failed to read xml: " .. (err1 or "unknown"))
				goto continue
			end

			local optsText, err2 = readExact(optsLen)
			if not optsText then
				sendErr("failed to read options: " .. (err2 or "unknown"))
				goto continue
			end

			io.stderr:write(string.format("[worker] FIND_BEST_ANOINT: xml=%d bytes, opts=%d bytes\n",
				xmlLen, optsLen))

			-- Parse options JSON
			local function parseAnointOpts(s)
				local result = {}
				-- Extract stat (sorting stat)
				local stat = s:match('"stat"%s*:%s*"([^"]+)"')
				result.stat = stat or "CombinedDPS"
				-- Extract max_results
				local maxResults = s:match('"max_results"%s*:%s*(%d+)')
				result.maxResults = tonumber(maxResults) or 30
				-- Extract search filter
				local search = s:match('"search"%s*:%s*"([^"]*)"')
				result.search = search or ""
				-- Extract slot_name override (default "Amulet")
				local slotName = s:match('"slot_name"%s*:%s*"([^"]*)"')
				result.slotName = slotName or "Amulet"
				return result
			end

			local opts = parseAnointOpts(optsText)

			io.stderr:write(string.format("[worker] FIND_BEST_ANOINT: stat=%q, maxResults=%d, slot=%q\n",
				opts.stat, opts.maxResults, opts.slotName))

			-- Step 1: Load build
			local ok, errMsg = pcall(function()
				loadBuildFromXML(xmlText, "find_best_anoint")
			end)
			if not ok then
				sendErr("loadBuildFromXML failed: " .. tostring(errMsg))
				goto continue
			end
			if not build or not build.calcsTab then
				sendErr("build or calcsTab not available after loading")
				goto continue
			end

			-- Ensure calculation is complete (buildFlag may be deferred)
			if build.buildFlag then
				for i = 1, 10 do
					runCallback("OnFrame")
					if not build.buildFlag then break end
				end
			end
			if not build.calcsTab.mainEnv then
				local buildOk, buildErr = pcall(function()
					build.calcsTab:BuildOutput()
				end)
				if not buildOk or not build.calcsTab.mainEnv then
					sendErr("BuildOutput failed for this build: " .. tostring(buildErr or "mainEnv is nil"))
					goto continue
				end
			end

			-- Step 2: Get the calculator
			local calcFunc, baseOutput = build.calcsTab:GetMiscCalculator()
			if not calcFunc then
				sendErr("GetMiscCalculator returned nil")
				goto continue
			end

			-- Step 3: Find the Amulet item (or specified slot) in itemsTab
			local itemsTab = build.itemsTab
			if not itemsTab then
				sendErr("build.itemsTab not available")
				goto continue
			end

			local targetSlot = nil
			for _, slot in pairs(itemsTab.slots) do
				if slot.slotName == opts.slotName then
					targetSlot = slot
					break
				end
			end

			if not targetSlot then
				sendErr("slot not found: " .. opts.slotName)
				goto continue
			end

			-- Get current amulet item
			local currentItem = targetSlot.selItemId and itemsTab.items[targetSlot.selItemId]

			-- Step 4: Create base anoint item (remove existing anoint) for comparison baseline
			-- We need to build an item creation function similar to ItemsTab:anointItem
			-- but without depending on GUI state (displayItem, anointEnchantSlot)
			local function makeAnointedItem(sourceItem, node)
				if not sourceItem then
					-- Create a bare amulet if no item equipped
					local raw = "Rarity: NORMAL\nAnoint Test Amulet\nPaua Amulet"
					if node then
						raw = raw .. "\nAllocates " .. node.dn
					end
					local item = new("Item", raw)
					return item
				end
				local item = new("Item", sourceItem:BuildRaw())
				item.id = sourceItem.id
				-- Remove existing anoint from enchantModLines
				local newEnchants = {}
				for _, modLine in ipairs(item.enchantModLines or {}) do
					local line = modLine.line or ""
					if not line:find("^Allocates ") then
						table.insert(newEnchants, modLine)
					end
				end
				item.enchantModLines = newEnchants
				-- Add new anoint if node provided
				if node then
					table.insert(item.enchantModLines, 1, { crafted = true, line = "Allocates " .. node.dn })
				end
				item:BuildAndParseRaw()
				return item
			end

			-- Calculate baseline (no anoint)
			local baseItem = makeAnointedItem(currentItem, nil)
			local calcBase = calcFunc({ repSlotName = opts.slotName, repItem = baseItem })

			-- Step 5: Determine which stat to use for comparison
			-- Use the powerStatList approach — find the right stat config
			local sortStat = opts.stat
			local sortTransform = nil

			-- Common stat transforms (from POB's powerStatList)
			local statTransforms = {
				-- Percentage-based stats that need inversion
			}

			-- Step 6: Iterate through all anoitable nodes in the passive tree
			local treeNodes = build.spec and build.spec.tree and build.spec.tree.nodes
			if not treeNodes then
				sendErr("passive tree nodes not available")
				goto continue
			end

			local results = {}
			local nodesTested = 0
			local nodesSkipped = 0
			local searchLower = opts.search:lower()

			-- Get the set of currently allocated nodes to mark them
			local allocatedNodes = {}
			if build.spec and build.spec.allocNodes then
				for nodeId, _ in pairs(build.spec.allocNodes) do
					allocatedNodes[nodeId] = true
				end
			end

			-- Get current anoints to mark
			local currentAnoints = {}
			if currentItem then
				for _, modList in ipairs({
					currentItem.enchantModLines or {},
					currentItem.implicitModLines or {},
					currentItem.explicitModLines or {},
				}) do
					for _, mod in ipairs(modList) do
						local line = mod.line or ""
						local anointName = line:match("^Allocates (.+)")
						if anointName then
							currentAnoints[anointName] = true
						end
					end
				end
			end

			for id, node in pairs(treeNodes) do
				-- Only process anoitable nodes (those with recipe data)
				if node.recipe and #node.recipe >= 1 and node.dn and node.dn ~= "" then
					-- Apply search filter if provided
					if searchLower ~= "" then
						local found = false
						if node.dn:lower():find(searchLower, 1, true) then
							found = true
						end
						if not found and node.sd then
							for _, line in ipairs(node.sd) do
								if line:lower():find(searchLower, 1, true) then
									found = true
									break
								end
							end
						end
						if not found then
							nodesSkipped = nodesSkipped + 1
							goto skip_anoint_node
						end
					end

					nodesTested = nodesTested + 1

					-- Create anointed item for this node
					local anointedItem = makeAnointedItem(currentItem, node)
					local output = calcFunc({ repSlotName = opts.slotName, repItem = anointedItem })

					-- Calculate power difference
					local baseVal, newVal
					if output.Minion and calcBase.Minion then
						baseVal = calcBase.Minion[sortStat] or calcBase[sortStat] or 0
						newVal = output.Minion[sortStat] or output[sortStat] or 0
					else
						baseVal = calcBase[sortStat] or 0
						newVal = output[sortStat] or 0
					end

					if sortTransform then
						baseVal = sortTransform(baseVal)
						newVal = sortTransform(newVal)
					end

					local diff = newVal - baseVal
					local isAllocated = allocatedNodes[node.id] or false
					local isCurrentAnoint = currentAnoints[node.dn] or false

					-- Collect node description
					local description = ""
					if node.sd then
						description = table.concat(node.sd, "\n")
					end

					-- Build oil recipe info
					local oilRecipe = {}
					if node.recipe then
						for _, oilIdx in ipairs(node.recipe) do
							table.insert(oilRecipe, oilIdx)
						end
					end

					table.insert(results, {
						nodeId = node.id or id,
						name = node.dn,
						description = description,
						diff = diff,
						newValue = newVal,
						baseValue = baseVal,
						isAllocated = isAllocated,
						isCurrentAnoint = isCurrentAnoint,
						oilRecipe = oilRecipe,
						isKeystone = node.isKeystone or false,
						isNotable = node.isNotable or false,
					})
				end
				::skip_anoint_node::
			end

			io.stderr:write(string.format("[worker] FIND_BEST_ANOINT: tested %d nodes, skipped %d\n",
				nodesTested, nodesSkipped))

			-- Step 7: Sort by diff descending
			table.sort(results, function(a, b)
				if a.diff == b.diff then
					return a.name < b.name
				end
				return a.diff > b.diff
			end)

			-- Step 8: Trim to max_results
			local trimmedResults = {}
			for i = 1, math.min(opts.maxResults, #results) do
				trimmedResults[i] = results[i]
			end

			-- Step 9: Build JSON result
			local resultParts = {}
			table.insert(resultParts, '{"stat":' .. jsonEncode(sortStat))
			table.insert(resultParts, ',"slot":' .. jsonEncode(opts.slotName))
			table.insert(resultParts, ',"current_item":' .. jsonEncode(currentItem and currentItem.name or ""))
			table.insert(resultParts, ',"nodes_tested":' .. jsonEncode(nodesTested))
			table.insert(resultParts, ',"nodes_skipped":' .. jsonEncode(nodesSkipped))
			table.insert(resultParts, ',"total_anointable":' .. jsonEncode(nodesTested + nodesSkipped))

			-- Current anoint info
			local currentAnointNames = {}
			for name, _ in pairs(currentAnoints) do
				table.insert(currentAnointNames, name)
			end
			table.insert(resultParts, ',"current_anoints":' .. jsonEncode(currentAnointNames))

			-- Encode results array
			local resParts = {}
			for _, r in ipairs(trimmedResults) do
				table.insert(resParts, jsonEncode({
					node_id = r.nodeId,
					name = r.name,
					description = r.description,
					diff = r.diff,
					new_value = r.newValue,
					base_value = r.baseValue,
					is_allocated = r.isAllocated,
					is_current_anoint = r.isCurrentAnoint,
					oil_recipe = r.oilRecipe,
					is_keystone = r.isKeystone,
					is_notable = r.isNotable,
				}))
			end
			table.insert(resultParts, ',"results":[' .. table.concat(resParts, ",") .. "]")
			table.insert(resultParts, "}")

			local resultJSON = table.concat(resultParts)

			io.stderr:write(string.format("[worker] FIND_BEST_ANOINT complete: %d results, top=%s (%.2f)\n",
				#trimmedResults,
				#trimmedResults > 0 and trimmedResults[1].name or "none",
				#trimmedResults > 0 and trimmedResults[1].diff or 0))

			sendOK(resultJSON)
			goto continue
		end
	end

	-- Parse RECALC command
	local inputLen = line:match("^RECALC (%d+)$")
	if not inputLen then
		sendErr("unknown command: " .. line)
		goto continue
	end

	inputLen = tonumber(inputLen)

	-- Read the XML payload
	local xmlText, readErr = readExact(inputLen)
	if not xmlText then
		sendErr("failed to read input: " .. (readErr or "unknown"))
		goto continue
	end

	io.stderr:write(string.format("[worker] Received RECALC request (%d bytes)\n", inputLen))

	-- Perform the recalculation
	local ok, errMsg = pcall(function()
		loadBuildFromXML(xmlText, "api_request")
	end)

	if not ok then
		sendErr("loadBuildFromXML failed: " .. tostring(errMsg))
		goto continue
	end

	if not build then
		sendErr("build object not available after loading")
		goto continue
	end

	if not build.calcsTab then
		sendErr("build calcsTab not available")
		goto continue
	end

	-- Diagnostic: log tree/class loading status to help debug compatibility issues
	if build.spec then
		local allocCount = 0
		if build.spec.allocNodes then
			for _ in pairs(build.spec.allocNodes) do allocCount = allocCount + 1 end
		end
		io.stderr:write(string.format("[worker] Tree diagnostic: treeVersion=%s, classId=%s, className=%s, ascendClassId=%s, ascendClassName=%s, allocNodes=%d\n",
			tostring(build.spec.treeVersion),
			tostring(build.spec.curClassId),
			tostring(build.spec.curClassName),
			tostring(build.spec.curAscendClassId),
			tostring(build.spec.curAscendClassName),
			allocCount))
		if allocCount == 0 then
			io.stderr:write("[worker] WARNING: No allocated tree nodes! Tree may have failed to load. Check if TreeData/" .. tostring(build.spec.treeVersion) .. "/tree.lua exists.\n")
		end
	else
		io.stderr:write("[worker] WARNING: build.spec is nil — passive tree not loaded\n")
	end

	-- Ensure calculation is complete: loadBuildFromXML may set buildFlag=true
	-- without finishing BuildOutput (e.g. when ElegantHubris data is missing,
	-- or complex builds with many Tattoos). We need OnFrame to trigger the
	-- deferred CalcsTab:BuildOutput() call.
	if build.buildFlag then
		io.stderr:write("[worker] buildFlag is set, running OnFrame to complete calculation...\n")
		for i = 1, 10 do
			runCallback("OnFrame")
			if not build.buildFlag then break end
		end
		if build.buildFlag then
			io.stderr:write("[worker] WARNING: buildFlag still set after 10 OnFrame calls\n")
		end
	end

	-- Verify mainEnv is populated (BuildOutput completed successfully)
	if not build.calcsTab.mainEnv then
		io.stderr:write("[worker] mainEnv is nil, attempting direct BuildOutput...\n")
		local buildOk, buildErr = pcall(function()
			build.calcsTab:BuildOutput()
		end)
		if not buildOk then
			sendErr("BuildOutput failed: " .. tostring(buildErr))
			goto continue
		end
		if not build.calcsTab.mainEnv then
			sendErr("BuildOutput completed but mainEnv is still nil — calculation failed for this build")
			goto continue
		end
	end

	-- Log some stats
	if build.calcsTab.mainOutput then
		local out = build.calcsTab.mainOutput
		local parts = {}
		if out.TotalDPS then table.insert(parts, string.format("DPS=%.0f", out.TotalDPS)) end
		if out.Life then table.insert(parts, string.format("Life=%.0f", out.Life)) end
		if out.EnergyShield then table.insert(parts, string.format("ES=%.0f", out.EnergyShield)) end
		if #parts > 0 then
			io.stderr:write("[worker] Stats: " .. table.concat(parts, ", ") .. "\n")
		end
	end

	-- Generate output XML by updating only the <Build> section in the original XML
	-- This preserves Tree, Items (with ModRange), Skills (including EnemyExplode),
	-- Config, and all other sections exactly as they were in the input.
	local saveOk, newXML, composeErr = pcall(function()
		return generateRecalcXML(xmlText)
	end)
	if not saveOk then
		sendErr("generateRecalcXML exception: " .. tostring(newXML))
		goto continue
	end
	if not newXML then
		sendErr("generateRecalcXML failed: " .. tostring(composeErr or "unknown error"))
		goto continue
	end

	io.stderr:write(string.format("[worker] Recalculation complete (%d bytes output)\n", #newXML))
	sendOK(newXML)

	::continue::
end
