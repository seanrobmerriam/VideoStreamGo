import React, { useState, useRef, useCallback } from 'react';

interface VideoUploaderProps {
  onUpload: (file: File, metadata: UploadMetadata) => Promise<void>;
  maxSize?: number; // in MB
  acceptedFormats?: string[];
}

interface UploadMetadata {
  title: string;
  description?: string;
  category?: string;
  tags?: string[];
}

interface UploadProgress {
  status: 'idle' | 'uploading' | 'processing' | 'complete' | 'error';
  progress: number;
  message?: string;
}

export default function VideoUploader({
  onUpload,
  maxSize = 500,
  acceptedFormats = ['video/mp4', 'video/webm', 'video/quicktime', 'video/x-matroska'],
}: VideoUploaderProps) {
  const [dragActive, setDragActive] = useState(false);
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<string | null>(null);
  const [progress, setProgress] = useState<UploadProgress>({ status: 'idle', progress: 0 });
  const [metadata, setMetadata] = useState<UploadMetadata>({ title: '', description: '' });
  const [error, setError] = useState<string | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const validateFile = (file: File): string | null => {
    if (!acceptedFormats.includes(file.type)) {
      return 'Invalid file format. Please upload MP4, WebM, QuickTime, or MKV.';
    }
    if (file.size > maxSize * 1024 * 1024) {
      return `File too large. Maximum size is ${maxSize}MB.`;
    }
    return null;
  };

  const handleDrag = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === 'dragenter' || e.type === 'dragover') {
      setDragActive(true);
    } else if (e.type === 'dragleave') {
      setDragActive(false);
    }
  }, []);

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);

    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
      const droppedFile = e.dataTransfer.files[0];
      const validationError = validateFile(droppedFile);
      
      if (validationError) {
        setError(validationError);
        return;
      }

      setFile(droppedFile);
      setError(null);
      
      // Create preview
      const url = URL.createObjectURL(droppedFile);
      setPreview(url);
    }
  }, []);

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      const selectedFile = e.target.files[0];
      const validationError = validateFile(selectedFile);
      
      if (validationError) {
        setError(validationError);
        return;
      }

      setFile(selectedFile);
      setError(null);
      
      const url = URL.createObjectURL(selectedFile);
      setPreview(url);
    }
  };

  const handleUpload = async () => {
    if (!file || !metadata.title) {
      setError('Please provide a title for your video.');
      return;
    }

    try {
      setProgress({ status: 'uploading', progress: 0, message: 'Uploading...' });
      
      // Simulate upload progress
      const progressInterval = setInterval(() => {
        setProgress((prev) => ({
          ...prev,
          progress: Math.min(prev.progress + 10, 90),
        }));
      }, 200);

      await onUpload(file, metadata);
      
      clearInterval(progressInterval);
      setProgress({ status: 'processing', progress: 95, message: 'Processing video...' });
      
      // Simulate processing completion
      setTimeout(() => {
        setProgress({ status: 'complete', progress: 100, message: 'Upload complete!' });
      }, 1500);
    } catch (err) {
      setProgress({ status: 'error', progress: 0, message: 'Upload failed. Please try again.' });
      setError('Failed to upload video. Please try again.');
    }
  };

  const resetUpload = () => {
    setFile(null);
    setPreview(null);
    setMetadata({ title: '', description: '' });
    setProgress({ status: 'idle', progress: 0 });
    setError(null);
    if (inputRef.current) {
      inputRef.current.value = '';
    }
  };

  return (
    <div className="space-y-6">
      {/* Drop Zone */}
      {!file && (
        <div
          className={`relative border-2 border-dashed rounded-xl p-8 text-center transition-colors ${
            dragActive
              ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20'
              : 'border-secondary-300 dark:border-secondary-700 hover:border-primary-400'
          }`}
          onDragEnter={handleDrag}
          onDragLeave={handleDrag}
          onDragOver={handleDrag}
          onDrop={handleDrop}
        >
          <input
            ref={inputRef}
            type="file"
            accept={acceptedFormats.join(',')}
            onChange={handleFileSelect}
            className="absolute inset-0 w-full h-full opacity-0 cursor-pointer"
          />
          
          <div className="flex flex-col items-center">
            <div className="h-16 w-16 rounded-full bg-primary-100 dark:bg-primary-900 flex items-center justify-center mb-4">
              <svg className="h-8 w-8 text-primary-600 dark:text-primary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
              </svg>
            </div>
            <p className="text-lg font-medium text-secondary-900 dark:text-white mb-1">
              Drag and drop your video here
            </p>
            <p className="text-sm text-secondary-500 dark:text-secondary-400 mb-4">
              or click to browse
            </p>
            <p className="text-xs text-secondary-400 dark:text-secondary-500">
              MP4, WebM, QuickTime, or MKV up to {maxSize}MB
            </p>
          </div>
        </div>
      )}

      {/* Error Message */}
      {error && (
        <div className="p-4 rounded-lg bg-red-50 border border-red-200 text-red-700 dark:bg-red-900/20 dark:border-red-800 dark:text-red-400">
          {error}
        </div>
      )}

      {/* File Preview & Metadata */}
      {file && preview && progress.status !== 'complete' && (
        <div className="grid gap-6 md:grid-cols-2">
          {/* Video Preview */}
          <div className="space-y-4">
            <div className="aspect-video bg-black rounded-lg overflow-hidden">
              <video src={preview} className="w-full h-full object-contain" controls />
            </div>
            <div className="flex items-center justify-between text-sm">
              <span className="text-secondary-500 dark:text-secondary-400">{file.name}</span>
              <span className="text-secondary-500 dark:text-secondary-400">
                {(file.size / (1024 * 1024)).toFixed(2)} MB
              </span>
            </div>
          </div>

          {/* Metadata Form */}
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-secondary-700 dark:text-secondary-300 mb-1">
                Title *
              </label>
              <input
                type="text"
                value={metadata.title}
                onChange={(e) => setMetadata({ ...metadata, title: e.target.value })}
                className="w-full rounded-lg border border-secondary-300 bg-white px-4 py-2.5 text-secondary-900 placeholder:text-secondary-400 focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500/20 dark:border-secondary-700 dark:bg-secondary-800 dark:text-secondary-100 dark:placeholder:text-secondary-500"
                placeholder="Enter video title"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-secondary-700 dark:text-secondary-300 mb-1">
                Description
              </label>
              <textarea
                value={metadata.description}
                onChange={(e) => setMetadata({ ...metadata, description: e.target.value })}
                rows={4}
                className="w-full rounded-lg border border-secondary-300 bg-white px-4 py-2.5 text-secondary-900 placeholder:text-secondary-400 focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500/20 dark:border-secondary-700 dark:bg-secondary-800 dark:text-secondary-100 dark:placeholder:text-secondary-500"
                placeholder="Describe your video"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-secondary-700 dark:text-secondary-300 mb-1">
                Category
              </label>
              <select
                value={metadata.category}
                onChange={(e) => setMetadata({ ...metadata, category: e.target.value })}
                className="w-full rounded-lg border border-secondary-300 bg-white px-4 py-2.5 text-secondary-900 focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500/20 dark:border-secondary-700 dark:bg-secondary-800 dark:text-secondary-100"
              >
                <option value="">Select category</option>
                <option value="javascript">JavaScript</option>
                <option value="typescript">TypeScript</option>
                <option value="react">React</option>
                <option value="nodejs">Node.js</option>
                <option value="devops">DevOps</option>
              </select>
            </div>

            <div className="flex gap-3 pt-4">
              <button
                onClick={resetUpload}
                className="flex-1 rounded-lg border border-secondary-300 bg-white px-4 py-2.5 text-sm font-medium text-secondary-700 hover:bg-secondary-50 focus:outline-none focus:ring-2 focus:ring-primary-500/20 dark:bg-secondary-800 dark:text-secondary-300 dark:border-secondary-600 dark:hover:bg-secondary-700"
              >
                Cancel
              </button>
              <button
                onClick={handleUpload}
                disabled={!metadata.title || progress.status === 'uploading'}
                className="flex-1 rounded-lg bg-primary-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-primary-500/20 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {progress.status === 'uploading' ? 'Uploading...' : 'Upload Video'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Upload Progress */}
      {progress.status !== 'idle' && progress.status !== 'complete' && (
        <div className="space-y-4">
          <div className="flex items-center justify-between text-sm">
            <span className="font-medium text-secondary-900 dark:text-white">
              {progress.message || (progress.status === 'uploading' ? 'Uploading...' : 'Processing...')}
            </span>
            <span className="text-secondary-500 dark:text-secondary-400">{progress.progress}%</span>
          </div>
          <div className="h-2 bg-secondary-200 dark:bg-secondary-700 rounded-full overflow-hidden">
            <div
              className="h-full bg-primary-500 rounded-full transition-all duration-300"
              style={{ width: `${progress.progress}%` }}
            />
          </div>
        </div>
      )}

      {/* Upload Complete */}
      {progress.status === 'complete' && (
        <div className="text-center py-8">
          <div className="h-16 w-16 rounded-full bg-green-100 text-green-600 flex items-center justify-center mx-auto mb-4">
            <svg className="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <h3 className="text-lg font-medium text-secondary-900 dark:text-white mb-2">
            Upload Complete!
          </h3>
          <p className="text-secondary-500 dark:text-secondary-400 mb-4">
            Your video has been uploaded and is now processing.
          </p>
          <button
            onClick={resetUpload}
            className="inline-flex items-center rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white hover:bg-primary-700"
          >
            Upload Another Video
          </button>
        </div>
      )}
    </div>
  );
}
