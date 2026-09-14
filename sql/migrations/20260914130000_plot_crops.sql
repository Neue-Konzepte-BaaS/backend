-- migrate:up
CREATE TABLE plot_crop (
    plot UUID NOT NULL,
    crop UUID NOT NULL,
    PRIMARY KEY (plot, crop),
    CONSTRAINT fk_plot_crop_plot
        FOREIGN KEY (plot)
        REFERENCES plot(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_plot_crop_crop
        FOREIGN KEY (crop)
        REFERENCES crop(id)
);

DROP TABLE field_crop;

-- migrate:down
CREATE TABLE field_crop (
    field UUID NOT NULL,
    crop UUID NOT NULL,
    PRIMARY KEY (field, crop),
    CONSTRAINT fk_field_crop_field
        FOREIGN KEY (field)
        REFERENCES field(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_field_crop_crop
        FOREIGN KEY (crop)
        REFERENCES crop(id)
);

DROP TABLE plot_crop;
