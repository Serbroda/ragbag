package de.serbroda.ragbag.controller.v1;

import de.serbroda.ragbag.dtos.page.CreatePageDto;
import de.serbroda.ragbag.dtos.page.PageDto;
import de.serbroda.ragbag.mappers.PageMapper;
import de.serbroda.ragbag.models.Page;
import de.serbroda.ragbag.services.PageService;
import de.serbroda.ragbag.utils.AuthorizationUtil;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.nio.file.AccessDeniedException;

@RestController
@RequestMapping("/api/v1/pages")
public class PageController {

    private final PageService pageService;

    public PageController(PageService pageService) {
        this.pageService = pageService;
    }

    @PostMapping
    public ResponseEntity<PageDto> createSpace(@RequestBody CreatePageDto dto) throws AccessDeniedException {
        Page entity = pageService.createPage(dto, AuthorizationUtil.getAuthenticatedAccountRequired());
        return ResponseEntity.ok(PageMapper.INSTANCE.map(entity));
    }
}
