import { useMemo, useState } from 'react';
import { 
  BarChart, Bar, LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer,
  PieChart, Pie, Cell
} from 'recharts';
import { BarChart2, TrendingUp, PieChart as PieChartIcon } from 'lucide-react';
import clsx from 'clsx';

interface ChartVisualizerProps {
  result: {
    columns: string[];
    rows: any[][];
  };
}

const COLORS = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899'];

export const ChartVisualizer = ({ result }: ChartVisualizerProps) => {
  const [chartType, setChartType] = useState<'bar' | 'line' | 'pie'>('bar');

  // Transform data for Recharts
  const data = useMemo(() => {
    if (!result || !result.rows || !result.columns) return [];
    return result.rows.map(row => {
      const obj: any = {};
      result.columns.forEach((col, i) => {
        obj[col] = row[i];
      });
      return obj;
    });
  }, [result]);

  // Auto-detect axes
  const { xAxis, yAxis, isChartable } = useMemo(() => {
    if (!result?.columns || result.columns.length < 2) {
      return { xAxis: '', yAxis: '', isChartable: false };
    }

    // Heuristic: Find first string/date column for X, first number column for Y
    let x = '';
    let y = '';

    // Check types based on first row
    if (result.rows.length > 0) {
      const firstRow = result.rows[0];
      
      // Find X (String or Date)
      for (let i = 0; i < result.columns.length; i++) {
        const val = firstRow[i];
        if (typeof val === 'string') {
           x = result.columns[i];
           break;
        }
      }

      // If no string found, use first column
      if (!x) x = result.columns[0];

      // Find Y (Number)
      for (let i = 0; i < result.columns.length; i++) {
        const col = result.columns[i];
        if (col === x) continue; // Don't use same col
        const val = firstRow[i];
        if (typeof val === 'number') {
          y = col;
          break;
        }
      }
    }

    return { xAxis: x, yAxis: y, isChartable: !!(x && y) };
  }, [result]);

  if (!isChartable) {
    return (
      <div className="p-4 text-center text-gray-500 text-sm italic">
        Cannot visualize this data as a chart (need at least one numeric column).
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-end gap-2">
        <button 
          onClick={() => setChartType('bar')}
          className={clsx("p-1.5 rounded hover:bg-white/10", chartType === 'bar' ? "bg-white/10 text-primary" : "text-gray-400")}
          title="Bar Chart"
        >
          <BarChart2 className="w-4 h-4" />
        </button>
        <button 
          onClick={() => setChartType('line')}
          className={clsx("p-1.5 rounded hover:bg-white/10", chartType === 'line' ? "bg-white/10 text-primary" : "text-gray-400")}
          title="Line Chart"
        >
          <TrendingUp className="w-4 h-4" />
        </button>
        <button 
          onClick={() => setChartType('pie')}
          className={clsx("p-1.5 rounded hover:bg-white/10", chartType === 'pie' ? "bg-white/10 text-primary" : "text-gray-400")}
          title="Pie Chart"
        >
          <PieChartIcon className="w-4 h-4" />
        </button>
      </div>

      <div className="h-64 w-full">
        <ResponsiveContainer width="100%" height="100%">
          {chartType === 'bar' ? (
            <BarChart data={data}>
              <CartesianGrid strokeDasharray="3 3" stroke="#374151" vertical={false} />
              <XAxis dataKey={xAxis} stroke="#9ca3af" tick={{fontSize: 12}} />
              <YAxis stroke="#9ca3af" tick={{fontSize: 12}} />
              <Tooltip 
                contentStyle={{ backgroundColor: '#1f2937', border: '1px solid #374151', borderRadius: '8px' }}
                itemStyle={{ color: '#e5e7eb' }}
              />
              <Legend />
              <Bar dataKey={yAxis} fill="#3b82f6" radius={[4, 4, 0, 0]} />
            </BarChart>
          ) : chartType === 'line' ? (
            <LineChart data={data}>
              <CartesianGrid strokeDasharray="3 3" stroke="#374151" vertical={false} />
              <XAxis dataKey={xAxis} stroke="#9ca3af" tick={{fontSize: 12}} />
              <YAxis stroke="#9ca3af" tick={{fontSize: 12}} />
              <Tooltip 
                contentStyle={{ backgroundColor: '#1f2937', border: '1px solid #374151', borderRadius: '8px' }}
                itemStyle={{ color: '#e5e7eb' }}
              />
              <Legend />
              <Line type="monotone" dataKey={yAxis} stroke="#3b82f6" strokeWidth={2} dot={{ fill: '#3b82f6' }} />
            </LineChart>
          ) : (
            <PieChart>
               <Pie
                data={data}
                cx="50%"
                cy="50%"
                labelLine={false}
                outerRadius={80}
                fill="#8884d8"
                dataKey={yAxis}
                nameKey={xAxis}
                label={({ name, percent }) => `${name} ${(percent! * 100).toFixed(0)}%`}
              >
                {data.map((_, index) => (
                  <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                ))}
              </Pie>
              <Tooltip 
                contentStyle={{ backgroundColor: '#1f2937', border: '1px solid #374151', borderRadius: '8px' }}
                itemStyle={{ color: '#e5e7eb' }}
              />
            </PieChart>
          )}
        </ResponsiveContainer>
      </div>
    </div>
  );
};
