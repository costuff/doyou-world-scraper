import { format } from 'date-fns';
import type { Job } from '../types';

interface JobCardProps {
  job: Job;
}

export const JobCard = ({ job }: JobCardProps) => {
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed': return 'bg-green-500';
      case 'running': return 'bg-blue-500';
      case 'failed': return 'bg-red-500';
      default: return 'bg-gray-500';
    }
  };

  const getStatusText = (status: string) => {
    return status.charAt(0).toUpperCase() + status.slice(1);
  };

  return (
    <div className="bg-slate-800 rounded-lg p-4 shadow-lg border border-slate-700">
      <div className="flex justify-between items-start mb-3">
        <div>
          <h3 className="text-lg font-semibold text-white">{job.type}</h3>
          <p className="text-sm text-slate-400">
            {format(new Date(job.created_at), 'MMM d, yyyy HH:mm:ss')}
          </p>
        </div>
        <span className={`px-3 py-1 rounded-full text-xs font-medium ${getStatusColor(job.status)} text-white`}>
          {getStatusText(job.status)}
        </span>
      </div>

      {job.status === 'running' && (
        <div className="mb-3">
          <div className="flex justify-between text-sm mb-1">
            <span className="text-slate-400">Progress</span>
            <span className="text-white font-medium">{job.progress}%</span>
          </div>
          <div className="w-full bg-slate-700 rounded-full h-2">
            <div
              className="bg-blue-500 h-2 rounded-full transition-all duration-300"
              style={{ width: `${job.progress}%` }}
            />
          </div>
        </div>
      )}

      {job.error && (
        <div className="mt-2 p-2 bg-red-900/20 border border-red-500 rounded text-sm text-red-300">
          {job.error}
        </div>
      )}

      {job.result && (
        <div className="mt-2 p-2 bg-slate-900 rounded text-sm text-slate-300">
          <pre className="whitespace-pre-wrap">{JSON.stringify(job.result, null, 2)}</pre>
        </div>
      )}

      <p className="text-xs text-slate-500 mt-2">Job ID: {job.id}</p>
    </div>
  );
};
