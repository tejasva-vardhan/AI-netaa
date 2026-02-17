# Department Routing – Test Guide

## Quick verification in browser

1. **Run the app:** `npm run dev` (from `frontend/`), then open the app and go to **Chat** (e.g. Dashboard → Register New Complaint).
2. Open **DevTools (F12) → Console**.
3. **Run all automated routing tests (no manual steps):**
   ```javascript
   runAllRoutingTests()
   ```
   You should see ✅/❌ for each of 6 scenarios and a summary: `Result: X passed, Y failed`.
4. **Or verify current form data:** After filling problem/location/department, run:
   ```javascript
   testRecipients()
   ```
   Returns: `{ recipients, area, department, problem, includesSDM, includesCollector, includesRuralEngg }`.

4. **On actual submit** (in dev), the console will also log:
   - `[Department Routing] Recipients: [...]`
   - `Area | Dept | Severity`

---

## Test cases and expected results

### TEST 1: Basic submission
- **Steps:** Login → Register New Complaint → Type "सड़क टूटी हुई है" → Location → Select any department → Photo → (optional) Voice → Submit.
- **Check:** Complaint submits; console shows recipients; `aineta502@gmail.com` is in the list.

### TEST 2: Shivpuri city (no SDM)
- **Location:** Shivpuri city (e.g. address containing "Shivpuri" / PIN 473551).
- **Problem:** "गंदगी फैली हुई है". Select **नगर पालिका (Municipal Council)**.
- **Expect:**
  - ✅ aineta502@gmail.com  
  - ✅ cmoshivpuri@mpurban.gov.in  
  - ❌ No SDM (Shivpuri city is excluded from SDM routing).

### TEST 3: Kolaras (SDM Kolaras)
- **Location:** Kolaras area (address containing "Kolaras" / 473770).
- **Problem:** "बिजली नहीं है". Select **बिजली विभाग (Electricity)**.
- **Expect:**
  - ✅ aineta502@gmail.com  
  - ✅ seshivpuri.cz@mp.gov.in  
  - ✅ sdmkolaras2013@gmail.com  
  - ✅ eeresshivpuri-mp@nic.in (Rural Engineering for rural tehsil).

### TEST 4: Keyword suggestion in dropdown
- **Problem:** "बिजली का खंभा गिरा हुआ है" → **बिजली विभाग** should appear at **top** of the list (keyword sort).
- **Problem:** "स्कूल में पानी नहीं" → **शिक्षा विभाग** and **जल संसाधन** (or PHED) should appear at **top** (both keyword-matched).

### TEST 5: Pohri rural (SDM Pohri + PWD + Rural Engg)
- **Location:** Pohri (address containing "Pohri" / 473793).
- **Problem:** "पुल टूटा हुआ". Select **लोक निर्माण विभाग (PWD)**.
- **Expect:**
  - ✅ aineta502@gmail.com  
  - ✅ eepwdshivpuri@mp.nic.in  
  - ✅ ropohari24@gmail.com (SDM Pohri)  
  - ✅ eeresshivpuri-mp@nic.in (Rural Engineering).

### TEST 6: Serious complaint (Collector)
- **Problem:** Include "बहुत गंभीर" or "जान का खतरा" (or "urgent"/"critical").
- **Expect:** Recipients include **dmshivpuri@nic.in** (Collector).

### TEST 7: Multiple keywords
- **Problem:** "सड़क टूटी और बिजली नहीं".
- **Expect:** Department list shows **PWD** and **बिजली विभाग** at the **top** (keyword-sorted).

### TEST 8: No department selected
- **Steps:** Problem + Location + **do not** select department → Photo → Voice → Submit.
- **Expect:** Recipients still include **aineta502@gmail.com** and any keyword-matched departments (e.g. PWD for "सड़क").

---

## Verification checklist

- [ ] All complaints submit successfully.
- [ ] **aineta502@gmail.com** is always in recipients.
- [ ] Kolaras complaints include **sdmkolaras2013@gmail.com**.
- [ ] Pohri complaints include **ropohari24@gmail.com** (SDM Pohri).
- [ ] Karera complaints include **sdmkarera13@gmail.com** (when location has Karera in address).
- [ ] Shivpuri **city** complaints have **no** SDM.
- [ ] Keywords correctly move matching departments to top of list.
- [ ] Serious wording adds **dmshivpuri@nic.in** (Collector).
- [ ] Multiple keywords show multiple departments at top.
- [ ] Console shows a clear recipients array (and area/dept/severity in dev).

---

## How area is set (for SDM / rural)

- **Area** is derived from the **reverse-geocode address** (Nominatim) when you tap “Share location”.
- If the address contains **Kolaras, Pohri, Karera, Pichhore, Narwar, Badarwas, Khaniadhana** → that **tehsil** is set; the matching **SDM** and **Rural Engineering** are added.
- If the address contains **Shivpuri** (and no other tehsil) → area is treated as Shivpuri **city** and **no SDM** is added.
- For manual testing without real GPS you can temporarily set area in the console (e.g. via store or by mocking the address) to simulate different tehsils.

---

## If a test fails

Note and report:

1. **Test case number** (e.g. TEST 3).
2. **Expected vs actual** (e.g. “Expected SDM Kolaras, but recipients were […]”).
3. **Console output** (full `testRecipients()` output and any `[Department Routing]` logs).
4. **Exact steps** (problem text, selected department, and how you set location / area if simulated).

---

## Summary of implementation (for reference)

- **Area from address:** `getAreaFromAddress(address)` in `src/utils/locationArea.js`; used when setting location after GPS + reverse geocode.
- **No SDM for Shivpuri city:** Router skips SDM when `location.area === 'shivpuri'`.
- **Rural Engineering:** Added for any non-Shivpuri tehsil (Kolaras, Pohri, Karera, Pichhore, etc.).
- **Serious complaints:** Problem text matching “बहुत गंभीर”, “जान का खतरा”, “urgent”, “critical”, etc. sets severity high and adds Collector.
- **Keyword sorting:** Department list is sorted so that departments matching `getDepartmentsByKeywords(problem)` appear at the top.
- **Console helper:** In **dev**, `testRecipients()` is available on `window` after the Chat page has loaded.
