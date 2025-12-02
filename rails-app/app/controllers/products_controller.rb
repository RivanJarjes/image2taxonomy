class ProductsController < ApplicationController
  def index
    @products = Product.order(created_at: :desc)
    
    respond_to do |format|
      format.html
    end
  end

  def show
    @product = Product.find(params[:id])
    
    respond_to do |format|
      format.html
      format.turbo_stream
    end
  end

  def new
    @product = Product.new
  end

  def create
    @product = Product.new(product_params)
    if @product.save
      # Get the local file path for the uploaded image
      image_path = ActiveStorage::Blob.service.path_for(@product.image.key)
      
      # Enqueue job with both product_id and image_path for Go worker
      ProductAnalysisJob.perform_async(@product.id, image_path)
      redirect_to @product, notice: "Product uploaded and analysis started"
    else
      render :new, status: :unprocessable_entity
    end
  end

  private

  def product_params
    params.require(:product).permit(:title, :image)
  end
end
