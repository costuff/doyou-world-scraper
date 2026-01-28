import { JobCard } from './JobCard';
import type { Job } from '../types';

interface JobsListProps {
  jobs: Record<string, Job>;
}

export const JobsList = ({ jobs }: JobsListProps) => {
  if (!jobs || typeof jobs !== 'object') {
    return null;
  }

  const jobsArray = Object.values(jobs)
    .filter(job => job && job.id && job.created_at)
    .sort((a, b) => {
      try {
        return new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
      } catch {
        return 0;
      }
    });

  if (jobsArray.length === 0) {
    return (
      <div className="bg-slate-800 rounded-lg p-6 shadow-lg border border-slate-700">
        <h2 className="text-2xl font-bold text-white mb-4">Active Jobs</h2>
        <p className="text-slate-400 text-center py-8">No jobs yet. Start a scraping or download task!</p>
      </div>
    );
  }

  return (
    <div className="bg-slate-800 rounded-lg p-6 shadow-lg border border-slate-700">
      <h2 className="text-2xl font-bold text-white mb-4">Active Jobs</h2>
      <div className="space-y-4 max-h-[600px] overflow-y-auto">
        {jobsArray.map((job) => (
          <JobCard key={job.id} job={job} />
        ))}
      </div>
    </div>
  );
};
