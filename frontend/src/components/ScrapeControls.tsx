import { useState } from 'react';
import { format } from 'date-fns';
import apiService from '../services/api';

export const ScrapeControls = () => {
  const [date, setDate] = useState(format(new Date(), 'yyyy-MM-dd'));
  const [month, setMonth] = useState(format(new Date(), 'yyyy-MM'));
  const [year, setYear] = useState(format(new Date(), 'yyyy'));
  const [loading, setLoading] = useState<string | null>(null);
  const [message, setMessage] = useState<{ type: 'success' | 'error', text: string } | null>(null);

  const handleScrapeDate = async () => {
    setLoading('date');
    setMessage(null);
    try {
      const response = await apiService.scrapeDate(date);
      setMessage({ type: 'success', text: `Scraping started! Job ID: ${response.data.job_id}` });
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Failed to start scraping' });
    } finally {
      setLoading(null);
    }
  };

  const handleScrapeMonth = async () => {
    setLoading('month');
    setMessage(null);
    try {
      const response = await apiService.scrapeMonth(month);
      setMessage({ type: 'success', text: `Scraping started! Job ID: ${response.data.job_id}` });
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Failed to start scraping' });
    } finally {
      setLoading(null);
    }
  };

  const handleScrapeYear = async () => {
    setLoading('year');
    setMessage(null);
    try {
      const response = await apiService.scrapeYear(year);
      setMessage({ type: 'success', text: `Scraping started! Job ID: ${response.data.job_id}` });
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Failed to start scraping' });
    } finally {
      setLoading(null);
    }
  };

  return (
    <div className="bg-slate-800 rounded-lg p-6 shadow-lg border border-slate-700">
      <h2 className="text-2xl font-bold text-white mb-6">Scrape Tracks</h2>
      
      {message && (
        <div className={`mb-4 p-3 rounded ${message.type === 'success' ? 'bg-green-900/20 border border-green-500 text-green-300' : 'bg-red-900/20 border border-red-500 text-red-300'}`}>
          {message.text}
        </div>
      )}

      <div className="space-y-4">
        {/* Scrape Date */}
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-2">
            Scrape Single Date
          </label>
          <div className="flex gap-2">
            <input
              type="date"
              value={date}
              onChange={(e) => setDate(e.target.value)}
              className="flex-1 px-3 py-2 bg-slate-900 border border-slate-700 rounded text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <button
              onClick={handleScrapeDate}
              disabled={loading === 'date'}
              className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-slate-600 text-white rounded font-medium transition-colors"
            >
              {loading === 'date' ? 'Scraping...' : 'Scrape Date'}
            </button>
          </div>
        </div>

        {/* Scrape Month */}
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-2">
            Scrape Entire Month
          </label>
          <div className="flex gap-2">
            <input
              type="month"
              value={month}
              onChange={(e) => setMonth(e.target.value)}
              className="flex-1 px-3 py-2 bg-slate-900 border border-slate-700 rounded text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <button
              onClick={handleScrapeMonth}
              disabled={loading === 'month'}
              className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-slate-600 text-white rounded font-medium transition-colors"
            >
              {loading === 'month' ? 'Scraping...' : 'Scrape Month'}
            </button>
          </div>
        </div>

        {/* Scrape Year */}
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-2">
            Scrape Entire Year
          </label>
          <div className="flex gap-2">
            <input
              type="number"
              value={year}
              onChange={(e) => setYear(e.target.value)}
              min="2020"
              max={new Date().getFullYear()}
              className="flex-1 px-3 py-2 bg-slate-900 border border-slate-700 rounded text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <button
              onClick={handleScrapeYear}
              disabled={loading === 'year'}
              className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-slate-600 text-white rounded font-medium transition-colors"
            >
              {loading === 'year' ? 'Scraping...' : 'Scrape Year'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
