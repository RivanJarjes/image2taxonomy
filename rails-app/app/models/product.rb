class Product < ApplicationRecord
  has_one_attached :image

  enum :processing_status, { pending: "pending", processing: "processing", complete: "complete", failed: "failed" }

  validates :image, presence: true
  before_create :set_default_title

  private

  def set_default_title
    self.title ||= image.filename.to_s.gsub(/\.[^.]+$/, '')
  end
end

