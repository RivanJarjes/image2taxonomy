class ProductAnalysisJob
  include Sidekiq::Job

  def perform(product_id, image_path)
    # This job is processed by the Go worker
    # The Go worker reads from the same Redis queue and handles the actual AI processing
    # Rails just enqueues the job with product_id and image_path
  end
end
