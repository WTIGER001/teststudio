export const statusWindows = [{ id: '24h', label: '24 hours' }, { id: '7d', label: '7 days' }, { id: '1m', label: '1 month (30 days)' }, { id: '1y', label: '1 year (365 days)' }, { id: 'all', label: 'All time' }]
export function statusTextColor(background: string): string {
  const channels = [1, 3, 5].map(offset => parseInt(background.slice(offset, offset + 2), 16) / 255).map(value => value <= .04045 ? value / 12.92 : ((value + .055) / 1.055) ** 2.4)
  return channels[0] * .2126 + channels[1] * .7152 + channels[2] * .0722 > .179 ? '#111827' : '#ffffff'
}


export function monitoredLabel(value: number): string { return value > 0 && value < .1 ? '<0.1%' : value.toFixed(1) + '%' }
