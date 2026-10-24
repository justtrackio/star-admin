import { Tag } from 'antd';

const stateColors: Record<string, string> = {
  RUNNING: 'green', NEED_SCHEDULE: 'blue', PAUSED: 'orange', UNSTABLE: 'orange', CANCELLED: 'red', STOPPED: 'default',
};

export function RoutineLoadStatus({ state }: { state: string }) {
  return <Tag color={stateColors[state] ?? 'default'}>{state || 'Unknown'}</Tag>;
}
