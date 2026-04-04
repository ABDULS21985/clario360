'use client';

import { LineChart, Line, ResponsiveContainer } from 'recharts';

interface SparkLineProps {
  data: number[];
  color?: string;
  width?: number;
  height?: number;
  strokeWidth?: number;
}

export function SparkLine({
  data,
  color = 'hsl(158, 59%, 25%)',
  width = 80,
  height = 28,
  strokeWidth = 2,
}: SparkLineProps) {
  if (!data || data.length < 2) return null;

  const chartData = data.map((value, index) => ({ index, value }));

  return (
    <div style={{ width, height }}>
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={chartData} margin={{ top: 2, right: 2, bottom: 2, left: 2 }}>
          <Line
            type="monotone"
            dataKey="value"
            stroke={color}
            strokeWidth={strokeWidth}
            dot={false}
            animationDuration={600}
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
