export function isKafkaOffsetError(reason: string) {
  return /offset out of range|requested offset not available/i.test(reason);
}
