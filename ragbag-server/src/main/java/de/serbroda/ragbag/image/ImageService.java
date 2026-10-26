package de.serbroda.ragbag.image;

import de.serbroda.ragbag.shared.http.SafeHttpClient;
import java.awt.Graphics2D;
import java.awt.RenderingHints;
import java.awt.image.BufferedImage;
import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.util.Arrays;
import java.util.Iterator;
import java.util.Optional;
import javax.imageio.ImageIO;
import javax.imageio.ImageReader;
import javax.imageio.stream.ImageInputStream;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.web.multipart.MultipartFile;

/**
 * Stores avatars and space icons. Uploads are re-encoded as square PNGs: that keeps them small, strips metadata
 * such as GPS positions, and guarantees we only ever serve real images.
 */
@Transactional
@RequiredArgsConstructor
@Service
public class ImageService {

    static final int MAX_UPLOAD_BYTES = 5 * 1024 * 1024;
    /** GIF (possibly animated) and WebP cannot be re-encoded without losing quality, so they are kept up to this. */
    static final int MAX_ORIGINAL_BYTES = 1024 * 1024;

    static final int SIZE = 256;
    /** Refuse decompression bombs before decoding: a tiny PNG can claim gigapixel dimensions. */
    private static final long MAX_PIXELS = 50_000_000L;

    private final ImageRepository imageRepository;
    private final SafeHttpClient httpClient;

    @Transactional(readOnly = true)
    public Optional<Image> findImage(String id) {
        return imageRepository.findById(id);
    }

    /** Stores the uploaded file, or downloads the image at {@code url} if no file was sent. */
    public Image store(MultipartFile file, String url) {
        byte[] bytes;
        if (file != null && !file.isEmpty()) {
            if (file.getSize() > MAX_UPLOAD_BYTES) {
                throw new IllegalArgumentException("Images must be smaller than 5 MB");
            }
            try {
                bytes = file.getBytes();
            } catch (IOException e) {
                throw new IllegalArgumentException("The upload could not be read", e);
            }
        } else if (url != null && !url.isBlank()) {
            String normalized = SafeHttpClient.requireHttpUrl(url);
            bytes = httpClient
                    .get(normalized, MAX_UPLOAD_BYTES)
                    .map(SafeHttpClient.Response::body)
                    .orElseThrow(
                            () -> new IllegalArgumentException(
                                    "The image could not be loaded from this URL (unreachable, not allowed or larger than 5 MB)"));
        } else {
            throw new IllegalArgumentException("Send an image file or the URL of one");
        }

        Image image = new Image();
        Normalized normalized = normalize(bytes);
        image.setContentType(normalized.contentType());
        image.setData(normalized.data());
        return imageRepository.save(image);
    }

    /** Null-safe, so callers can pass the previous image id of a user or space as is. */
    public void deleteImage(String id) {
        if (id != null) {
            imageRepository.deleteById(id);
        }
    }

    static Normalized normalize(byte[] bytes) {
        Format format = Format.detect(bytes);
        if (format == null) {
            throw new IllegalArgumentException("Only PNG, JPEG, GIF and WebP images are supported");
        }
        if ((format == Format.GIF || format == Format.WEBP) && bytes.length <= MAX_ORIGINAL_BYTES) {
            return new Normalized(format.contentType, bytes);
        }
        if (format == Format.WEBP) {
            throw new IllegalArgumentException("WebP images must be smaller than 1 MB");
        }
        return new Normalized("image/png", toSquarePng(decode(bytes)));
    }

    private static BufferedImage decode(byte[] bytes) {
        try (ImageInputStream input = ImageIO.createImageInputStream(new ByteArrayInputStream(bytes))) {
            Iterator<ImageReader> readers = ImageIO.getImageReaders(input);
            if (!readers.hasNext()) {
                throw new IllegalArgumentException("The image could not be read");
            }
            ImageReader reader = readers.next();
            try {
                reader.setInput(input, true, true);
                if ((long) reader.getWidth(0) * reader.getHeight(0) > MAX_PIXELS) {
                    throw new IllegalArgumentException("The image is too large");
                }
                return reader.read(0);
            } finally {
                reader.dispose();
            }
        } catch (IOException | RuntimeException e) {
            if (e instanceof IllegalArgumentException illegal) {
                throw illegal;
            }
            throw new IllegalArgumentException("The image could not be read", e);
        }
    }

    /** Center-crops to a square and scales down to at most {@link #SIZE} pixels (never up). */
    private static byte[] toSquarePng(BufferedImage source) {
        int side = Math.min(source.getWidth(), source.getHeight());
        BufferedImage current =
                source.getSubimage((source.getWidth() - side) / 2, (source.getHeight() - side) / 2, side, side);

        // Halving step by step keeps downscaled photos from looking grainy
        int target = Math.min(side, SIZE);
        int size = side;
        do {
            size = Math.max(size / 2, target);
            current = scale(current, size);
        } while (size > target);

        ByteArrayOutputStream output = new ByteArrayOutputStream();
        try {
            ImageIO.write(current, "png", output);
        } catch (IOException e) {
            throw new IllegalStateException("Failed to encode image", e);
        }
        return output.toByteArray();
    }

    private static BufferedImage scale(BufferedImage source, int size) {
        BufferedImage scaled = new BufferedImage(size, size, BufferedImage.TYPE_INT_ARGB);
        Graphics2D graphics = scaled.createGraphics();
        try {
            graphics.setRenderingHint(RenderingHints.KEY_INTERPOLATION, RenderingHints.VALUE_INTERPOLATION_BICUBIC);
            graphics.setRenderingHint(RenderingHints.KEY_RENDERING, RenderingHints.VALUE_RENDER_QUALITY);
            graphics.drawImage(source, 0, 0, size, size, null);
        } finally {
            graphics.dispose();
        }
        return scaled;
    }

    record Normalized(String contentType, byte[] data) {}

    /** Detected from the file content: the declared content type and file name are not trustworthy. */
    enum Format {
        PNG("image/png"),
        JPEG("image/jpeg"),
        GIF("image/gif"),
        WEBP("image/webp");

        final String contentType;

        Format(String contentType) {
            this.contentType = contentType;
        }

        static Format detect(byte[] bytes) {
            if (startsWith(bytes, 0, new byte[] {(byte) 0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})) {
                return PNG;
            }
            if (startsWith(bytes, 0, new byte[] {(byte) 0xff, (byte) 0xd8, (byte) 0xff})) {
                return JPEG;
            }
            if (startsWith(bytes, 0, "GIF87a".getBytes()) || startsWith(bytes, 0, "GIF89a".getBytes())) {
                return GIF;
            }
            if (startsWith(bytes, 0, "RIFF".getBytes()) && startsWith(bytes, 8, "WEBP".getBytes())) {
                return WEBP;
            }
            return null;
        }

        private static boolean startsWith(byte[] bytes, int offset, byte[] prefix) {
            return bytes.length >= offset + prefix.length
                    && Arrays.equals(bytes, offset, offset + prefix.length, prefix, 0, prefix.length);
        }
    }
}
