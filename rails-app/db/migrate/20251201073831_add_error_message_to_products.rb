class AddErrorMessageToProducts < ActiveRecord::Migration[8.1]
  def change
    add_column :products, :error_message, :text
  end
end
