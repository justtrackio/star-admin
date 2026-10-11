function pad(part: number): string {
  return String(part).padStart(2, '0');
}

export function formatTimestamp(value: string | number | null | undefined): string {
  if (value == null || value === '') return '—';

  // Preserve timezone-less cluster timestamps as reported by StarRocks.
  if (typeof value === 'string') {
    const clusterTime = /^(\d{4}-\d{2}-\d{2})[ T](\d{2}:\d{2}:\d{2})(?:\.\d+)?$/.exec(value);
    if (clusterTime) return `${clusterTime[1]} ${clusterTime[2]}`;
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '—';
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
}
