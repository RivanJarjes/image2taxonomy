class CreateProducts < ActiveRecord::Migration[8.1]
  def change
    create_table :products do |t|
      t.string :title
      t.text :description
      t.string :taxonomy
      t.string :processing_status, default: "pending"
      t.jsonb :violations, default: {}

      t.timestamps
    end
  end
end
