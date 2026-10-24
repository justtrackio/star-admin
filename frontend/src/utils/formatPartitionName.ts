export function formatPartitionName(name: string): string {
  return name.replace(/^p(\d{4})(\d{2})(\d{2})(\d{2})(\d{2})(\d{2})$/, '$1-$2-$3 $4:$5:$6Z');
}
