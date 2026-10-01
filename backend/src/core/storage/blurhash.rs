/// Decode a raster image and encode a 4×3 blurhash.
///
/// Returns the hash plus the original pixel size. Non-raster bytes return `None`.
pub fn blurhash_from_bytes(bytes: &[u8]) -> Option<(String, i32, i32)> {
    let image = image::load_from_memory(bytes).ok()?;
    let width = i32::try_from(image.width()).ok()?;
    let height = i32::try_from(image.height()).ok()?;
    if width <= 0 || height <= 0 {
        return None;
    }
    let long_edge = width.max(height);
    let (target_w, target_h) = if long_edge > 64 {
        let scale = 64.0 / f64::from(long_edge);
        (
            (f64::from(width) * scale).round().max(1.0) as u32,
            (f64::from(height) * scale).round().max(1.0) as u32,
        )
    } else {
        (image.width().max(1), image.height().max(1))
    };
    let small = image.resize_exact(target_w, target_h, image::imageops::FilterType::Triangle);
    let rgba = small.to_rgba8();
    let (w, h) = rgba.dimensions();
    let x_components = 4.min(w).max(1);
    let y_components = 3.min(h).max(1);
    let hash = blurhash::encode(x_components, y_components, w, h, rgba.as_raw()).ok()?;
    Some((hash, width, height))
}

#[cfg(test)]
mod tests {
    use super::blurhash_from_bytes;

    #[test]
    fn solid_red_png_has_a_stable_blurhash() {
        let image = image::RgbaImage::from_pixel(8, 8, image::Rgba([220, 40, 40, 255]));
        let mut bytes = Vec::new();
        image::DynamicImage::ImageRgba8(image)
            .write_to(
                &mut std::io::Cursor::new(&mut bytes),
                image::ImageFormat::Png,
            )
            .unwrap();
        let (hash, width, height) = blurhash_from_bytes(&bytes).unwrap();
        assert_eq!(width, 8);
        assert_eq!(height, 8);
        assert_eq!(hash, "LTPJVz|_fQ|_|_sofQsofQfQfQfQ");
    }

    #[test]
    fn text_bytes_have_no_blurhash() {
        assert!(blurhash_from_bytes(b"not an image").is_none());
    }
}
