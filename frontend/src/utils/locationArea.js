import { areaTehsils } from '../data/shivpuriDepartments';

/**
 * Derive tehsil/area from Nominatim address for SDM routing.
 * Returns area id (e.g. 'kolaras', 'pohri') or 'shivpuri' for Shivpuri city.
 * Shivpuri city: we set 'shivpuri' so router can skip SDM for city complaints.
 */
export function getAreaFromAddress(address) {
  if (!address || typeof address !== 'string') return undefined;
  const lower = address.toLowerCase();
  // Check non-Shivpuri tehsils first so "Kolaras, Shivpuri district" -> kolaras
  const tehsilOrder = ['kolaras', 'pohri', 'karera', 'pichhore', 'narwar', 'badarwas', 'khaniadhana'];
  for (const id of tehsilOrder) {
    const tehsil = areaTehsils.find((t) => t.id === id);
    if (!tehsil) continue;
    const enPart = tehsil.name.match(/\(([^)]+)\)/)?.[1]?.toLowerCase() || '';
    if (lower.includes(tehsil.id) || (enPart && lower.includes(enPart))) return tehsil.id;
  }
  if (lower.includes('shivpuri')) return 'shivpuri';
  return undefined;
}
