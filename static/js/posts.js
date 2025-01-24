const ImageHandler = (function() {
    let quill;
    
    let isResizing = false
    let imgElement = null
    let imgWidth = 0
    let imgHeight = 0
    let aspectRatio = 0
    let startX = 0
    let startY = 0

    const imageHandler = () => {
        const input = document.createElement('input')
        input.setAttribute('type', 'file')
        input.setAttribute('accept', 'image/*')
        input.onchange = async () => {
            const file = input.files[0]
            const formData = new FormData()
            formData.append('image', file)

            try {
                const response = await fetch('/post/upload-image', {
                    method: 'POST',
                    body: formData
                })
                const data = await response.json()

                const range = quill.getSelection()
                quill.insertEmbed(range.index, 'image', data.url)
            } catch (err) {
                console.error('Image upload failed:', err)
            }
        };
        input.click()
    }

    const makeImagesResizable = () => {
        const images = document.querySelectorAll('#editor-container img')
        images.forEach(img => {
            if (!img.classList.contains('resizeable')) {
                img.classList.add('resizable')
                img.addEventListener('mousedown', startResizing)
            }
        })
    }

    const startResizing = (e) => {
        isResizing = true
        imgElement = e.target
        imgWidth = imgElement.offsetWidth
        imgHeight = imgElement.offsetHeight
        startX = e.clientX
        startY = e.clientY

        e.preventDefault()

        window.addEventListener('mousemove', resizeImage)
        window.addEventListener('mouseup', stopResizing)
    }

    const resizeImage = (e) => {
        if (!isResizing) return

        const dx = e.clientX - startX
        const dy = e.clientY - startY

        const newWidth = imgWidth + dx
        const newHeight = imgHeight + dy

        const proportionalHeight = newWidth / aspectRatio

        if (newWidth > 50 && proportionalHeight > 50) {
            imgElement.style.width = `${newWidth}px`
            imgElement.style.height = `${proportionalHeight}px`
        }
    }

    const stopResizing = () => {
        isResizing = false
        window.removeEventListener('mousemove', resizeImage)
        window.removeEventListener('mouseup', stopResizing)
    }

    return {
        // setup
        setQuill: (q) => quill = q,

        // usage
        imageHandler: imageHandler,
        makeImagesResizable: makeImagesResizable,
    }
})()

document.addEventListener('DOMContentLoaded', function () {
    const quill = new Quill('#editor-container', {
        theme: 'snow',
        modules: {
            toolbar: {
                container: [
                    ['bold', 'italic', 'underline', 'strike'],
                    ['image']
                ],
                handlers: {
                    image: ImageHandler.imageHandler
                }
            }
        }
    });

    ImageHandler.setQuill(quill)

    quill.root.innerHTML = document.getElementById('quill-content').value
    ImageHandler.makeImagesResizable()
    quill.on('editor-change', () => ImageHandler.makeImagesResizable());

    document.getElementById('add-post-form').addEventListener('submit', function () {
        var quillContent = document.getElementById('quill-content')
        quillContent.value = quill.root.innerHTML
    })
})